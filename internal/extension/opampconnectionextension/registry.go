// Copyright  observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This file is roughly equivalent to registry.go in the opampextension
// package of opentelemetry-collector-contrib
// (github.com/open-telemetry/opentelemetry-collector-contrib/extension/opampextension).
// It is kept deliberately close to that file so that the two
// implementations can be compared and kept in sync. The main difference is
// that the underlying OpAMP client is supplied after construction via
// setClient, because in this distribution the OpAMP connection is owned by
// the opamp package rather than the extension itself.

package opampconnectionextension

import (
	"container/list"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/open-telemetry/opamp-go/protobufs"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/opampcustommessages"
)

// Client is the subset of the OpAMP client used by customCapabilityRegistry.
// It is exported so that the owner of the OpAMP connection can implement it
// to intercept calls (for example to merge in additional capabilities)
// before delegating to the real client.
type Client interface {
	SetCustomCapabilities(customCapabilities *protobufs.CustomCapabilities) error
	SendCustomMessage(message *protobufs.CustomMessage) (messageSendingChannel chan struct{}, err error)
}

type customCapabilityRegistry struct {
	mux                     *sync.Mutex
	capabilityToMsgChannels map[string]*list.List
	client                  Client
	logger                  *zap.Logger
}

var _ opampcustommessages.CustomCapabilityRegistry = (*customCapabilityRegistry)(nil)

func newCustomCapabilityRegistry(logger *zap.Logger) *customCapabilityRegistry {
	return &customCapabilityRegistry{
		mux:                     &sync.Mutex{},
		capabilityToMsgChannels: make(map[string]*list.List),
		logger:                  logger,
	}
}

// setClient supplies the underlying OpAMP client to the registry. Any
// capabilities that were Registered before the client was supplied are
// advertised to the server now.
func (cr *customCapabilityRegistry) setClient(c Client) {
	cr.mux.Lock()
	defer cr.mux.Unlock()
	cr.client = c
	if caps := cr.capabilities(); len(caps) > 0 {
		if err := c.SetCustomCapabilities(&protobufs.CustomCapabilities{
			Capabilities: caps,
		}); err != nil {
			// Not fatal: the server just won't know we support these
			// capabilities until something else triggers an update.
			cr.logger.Error("Failed to advertise capabilities after client was set", zap.Error(err))
		}
	}
}

// Register implements CustomCapabilityRegistry.Register.
//
// Register tolerates being called before setClient: in that case the
// capability is recorded and will be advertised to the OpAMP server once the
// client is supplied. This matters in the v1 agent, where components (e.g.
// the snapshot processor) register their capabilities during pipeline
// startup, but the owner of the OpAMP connection only calls setClient after
// the collector has finished starting.
func (cr *customCapabilityRegistry) Register(capability string, opts ...opampcustommessages.CustomCapabilityRegisterOption) (opampcustommessages.CustomCapabilityHandler, error) {
	optsStruct := opampcustommessages.DefaultCustomCapabilityRegisterOptions()
	for _, opt := range opts {
		opt(optsStruct)
	}

	cr.mux.Lock()
	defer cr.mux.Unlock()

	capabilityList := cr.capabilityToMsgChannels[capability]
	if capabilityList == nil {
		capabilityList = list.New()
		cr.capabilityToMsgChannels[capability] = capabilityList
	}

	msgChan := make(chan *protobufs.CustomMessage, optsStruct.MaxQueuedMessages)
	callbackElem := capabilityList.PushBack(msgChan)

	// If the client is already set, advertise the new capability immediately.
	// Otherwise setClient will flush the accumulated set when it's called.
	if cr.client != nil {
		capabilities := cr.capabilities()
		if !slices.Contains(capabilities, capability) {
			capabilities = append(capabilities, capability)
		}
		if err := cr.client.SetCustomCapabilities(&protobufs.CustomCapabilities{
			Capabilities: capabilities,
		}); err != nil {
			// Roll back so a failed advertisement does not leave a
			// dangling registration.
			capabilityList.Remove(callbackElem)
			if capabilityList.Front() == nil {
				delete(cr.capabilityToMsgChannels, capability)
			}
			return nil, fmt.Errorf("set custom capabilities: %w", err)
		}
	}

	unregisterFunc := cr.removeCapabilityFunc(capability, callbackElem)
	sender := newCustomMessageHandler(cr, capability, msgChan, unregisterFunc)

	return sender, nil
}

// ProcessMessage processes a custom message, asynchronously broadcasting it to all registered capability handlers for
// the messages capability.
func (cr customCapabilityRegistry) ProcessMessage(cm *protobufs.CustomMessage) {
	cr.mux.Lock()
	defer cr.mux.Unlock()

	msgChannels, ok := cr.capabilityToMsgChannels[cm.Capability]
	if !ok {
		return
	}

	for node := msgChannels.Front(); node != nil; node = node.Next() {
		msgChan, ok := node.Value.(chan *protobufs.CustomMessage)
		if !ok {
			continue
		}

		// If the channel is full, we will skip sending the message to the receiver.
		// We do this because we don't want a misbehaving component to be able to
		// block the opamp extension, or block other components from receiving messages.
		select {
		case msgChan <- cm:
		default:
		}
	}
}

// removeCapabilityFunc returns a func that removes the custom capability with the given msg channel list element and sender,
// then recalculates and sets the list of custom capabilities on the OpAMP client.
func (cr *customCapabilityRegistry) removeCapabilityFunc(capability string, callbackElement *list.Element) func() {
	return func() {
		cr.mux.Lock()
		defer cr.mux.Unlock()

		msgChanList := cr.capabilityToMsgChannels[capability]
		msgChanList.Remove(callbackElement)

		if msgChanList.Front() == nil {
			// Since there are no more callbacks for this capability,
			// this capability is no longer supported
			delete(cr.capabilityToMsgChannels, capability)
		}

		if cr.client == nil {
			return
		}

		capabilities := cr.capabilities()
		err := cr.client.SetCustomCapabilities(&protobufs.CustomCapabilities{
			Capabilities: capabilities,
		})
		if err != nil {
			// It's OK if we couldn't actually remove the capability, it just means we won't
			// notify the server properly, and the server may send us messages that we have no associated callbacks for.
			cr.logger.Error("Failed to set new capabilities", zap.Error(err))
		}
	}
}

// capabilities gives the current set of custom capabilities with at least one
// callback registered.
func (cr *customCapabilityRegistry) capabilities() []string {
	return maps.Keys(cr.capabilityToMsgChannels)
}

type customMessageHandler struct {
	// unregisteredMux protects unregistered, and makes sure that a message cannot be sent
	// on an unregistered capability.
	unregisteredMux *sync.Mutex

	capability               string
	registry                 *customCapabilityRegistry
	sendChan                 <-chan *protobufs.CustomMessage
	unregisterCapabilityFunc func()

	unregistered bool
}

var _ opampcustommessages.CustomCapabilityHandler = (*customMessageHandler)(nil)

func newCustomMessageHandler(
	registry *customCapabilityRegistry,
	capability string,
	sendChan <-chan *protobufs.CustomMessage,
	unregisterCapabilityFunc func(),
) *customMessageHandler {
	return &customMessageHandler{
		unregisteredMux: &sync.Mutex{},

		capability:               capability,
		registry:                 registry,
		sendChan:                 sendChan,
		unregisterCapabilityFunc: unregisterCapabilityFunc,
	}
}

// Message implements CustomCapabilityHandler.Message
func (c *customMessageHandler) Message() <-chan *protobufs.CustomMessage {
	return c.sendChan
}

// SendMessage implements CustomCapabilityHandler.SendMessage. The client is
// looked up through the registry at send time so that a handler obtained
// before the client was supplied still works once setClient has been called.
func (c *customMessageHandler) SendMessage(messageType string, message []byte) (messageSendingChannel chan struct{}, err error) {
	c.unregisteredMux.Lock()
	defer c.unregisteredMux.Unlock()

	if c.unregistered {
		return nil, errors.New("capability has already been unregistered")
	}

	c.registry.mux.Lock()
	client := c.registry.client
	c.registry.mux.Unlock()

	if client == nil {
		return nil, ErrClientNotSet
	}

	cm := &protobufs.CustomMessage{
		Capability: c.capability,
		Type:       messageType,
		Data:       message,
	}

	return client.SendCustomMessage(cm)
}

// Unregister implements CustomCapabilityHandler.Unregister
func (c *customMessageHandler) Unregister() {
	c.unregisteredMux.Lock()
	defer c.unregisteredMux.Unlock()

	c.unregistered = true

	c.unregisterCapabilityFunc()
}
