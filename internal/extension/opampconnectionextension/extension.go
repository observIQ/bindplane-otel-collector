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

package opampconnectionextension

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/opampcustommessages"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.uber.org/zap"
)

// ErrClientNotSet is returned by operations that require the underlying
// OpAMP client (currently only sending messages) when they are attempted
// before the owner of the connection has called SetClient. Register itself
// does not return this error — capabilities Registered before the client is
// set are advertised automatically once SetClient is called.
var ErrClientNotSet = errors.New("opamp client has not been set on opamp connection extension")

// Registry is the public interface that processors and other collector
// components may retrieve from the host in order to register custom
// capabilities and send/receive custom messages over the shared OpAMP
// connection.
//
// It is a superset of opampcustommessages.CustomCapabilityRegistry.
//
// The owner of the OpAMP connection (the opamp package) is expected to
// supply the underlying client by calling SetClient and to forward incoming
// custom messages by calling ProcessMessage.
type Registry interface {
	opampcustommessages.CustomCapabilityRegistry

	// SetClient supplies the underlying OpAMP client that the registry will
	// use to advertise custom capabilities and send custom messages. Any
	// capabilities that were Registered before SetClient is called are
	// advertised to the server at this point.
	SetClient(c CustomCapabilityClient)

	// ProcessMessage dispatches a custom message received from the OpAMP
	// server to all handlers registered for its capability. Messages for
	// capabilities with no registered handler are silently dropped.
	ProcessMessage(cm *protobufs.CustomMessage)
}

// GetRegistry returns the Registry for the currently-started
// opamp_connection extension, or nil if no extension is started. Only one
// opamp_connection extension may be configured per collector, so this is
// effectively a singleton lookup.
//
// It is intended to be called by the owner of the OpAMP connection (the
// opamp package) so that it can wire its client into the extension once
// both have been created.
func GetRegistry() Registry {
	instanceMux.Lock()
	defer instanceMux.Unlock()
	if instance == nil {
		return nil
	}
	return instance
}

var (
	instanceMux sync.Mutex
	instance    *opampConnectionExtension
)

// opampConnectionExtension is the concrete extension implementation. It is a
// thin wrapper around customCapabilityRegistry that adds the extension
// lifecycle and a package-level singleton so the OpAMP connection owner can
// supply the underlying client.
type opampConnectionExtension struct {
	id       component.ID
	logger   *zap.Logger
	registry *customCapabilityRegistry
}

var (
	_ extension.Extension = (*opampConnectionExtension)(nil)
	_ Registry            = (*opampConnectionExtension)(nil)
)

func newExtension(id component.ID, logger *zap.Logger) *opampConnectionExtension {
	return &opampConnectionExtension{
		id:       id,
		logger:   logger,
		registry: newCustomCapabilityRegistry(logger),
	}
}

// Start records the extension as the package-level singleton so that the
// opamp package can look it up and supply the OpAMP client.
//
// Only a single opamp_connection extension may be configured per collector,
// since they would otherwise overwrite each other's advertised custom
// capabilities whenever a component registers or unregisters one.
func (e *opampConnectionExtension) Start(_ context.Context, _ component.Host) error {
	instanceMux.Lock()
	defer instanceMux.Unlock()
	if instance != nil {
		return fmt.Errorf(
			"only one opamp_connection extension may be configured per collector; %q is already configured",
			instance.id,
		)
	}
	instance = e
	return nil
}

// Shutdown clears the package-level singleton.
func (e *opampConnectionExtension) Shutdown(_ context.Context) error {
	instanceMux.Lock()
	defer instanceMux.Unlock()
	if instance == e {
		instance = nil
	}
	return nil
}

// SetClient implements Registry.
func (e *opampConnectionExtension) SetClient(c CustomCapabilityClient) {
	e.registry.setClient(c)
}

// Register implements opampcustommessages.CustomCapabilityRegistry.
func (e *opampConnectionExtension) Register(capability string, opts ...opampcustommessages.CustomCapabilityRegisterOption) (opampcustommessages.CustomCapabilityHandler, error) {
	return e.registry.Register(capability, opts...)
}

// ProcessMessage implements Registry.
func (e *opampConnectionExtension) ProcessMessage(cm *protobufs.CustomMessage) {
	e.registry.ProcessMessage(cm)
}
