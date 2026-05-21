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
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/hostcapabilities"
	"go.uber.org/zap"
)

// ErrClientNotSet is returned by operations that require the underlying
// OpAMP client (currently only sending messages) when they are attempted
// before the owner of the connection has called SetClient. Register itself
// does not return this error — capabilities Registered before the client is
// set are advertised automatically once SetClient is called.
var ErrClientNotSet = errors.New("opamp client has not been set on opamp connection extension")

// Registry is the bridge that the owner of the OpAMP connection (the opamp
// package) uses to wire its client into the opamp_connection extension and
// to forward incoming custom messages.
//
// Registry is a package-level singleton and outlives individual extension
// instances. The OpAMP client supplied via SetClient is cached and
// forwarded to whichever opamp_connection extension instance is currently
// attached, including instances that attach later (e.g. across a
// collector restart). Per-extension state — registered capabilities and
// message channels — lives with the extension instance and is abandoned
// when that instance shuts down.
//
// The Register half of the upstream
// opampcustommessages.CustomCapabilityRegistry interface is intentionally
// not exposed here; components retrieve the extension itself via the
// collector host and call Register on it directly.
type Registry interface {
	// SetClient supplies the underlying OpAMP client. The client is
	// cached and forwarded to the currently-attached extension instance
	// (if any) as well as to any instance that attaches in the future.
	SetClient(c Client)

	// ProcessMessage dispatches a custom message received from the OpAMP
	// server to all handlers registered with the currently-attached
	// extension instance. Messages received while no extension is
	// attached are silently dropped.
	ProcessMessage(cm *protobufs.CustomMessage)

	// ModuleInfos returns the collector module information captured from
	// the host that started the currently-attached extension instance
	// (via the hostcapabilities.ModuleInfo interface). Returns a zero
	// value if no extension is attached, the host did not implement the
	// interface, or no module info was provided. Used by the OpAMP
	// client owner to build an AvailableComponents report.
	ModuleInfos() service.ModuleInfos
}

// GetRegistry returns the package-level Registry. It is always non-nil,
// including when no opamp_connection extension is currently started:
// SetClient calls are cached and ProcessMessage calls are dropped in that
// case.
func GetRegistry() Registry {
	return &manager
}

// manager is the package-level bridge between the opamp package and
// whichever opamp_connection extension instance is currently started.
// Because the collector rebuilds every component from its factory on each
// restart, the extension instance itself is short-lived; the manager
// remembers the client across those restarts so capabilities registered
// by the next instance can be advertised immediately.
var manager instanceManager

type instanceManager struct {
	mux         sync.Mutex
	instance    *opampConnectionExtension
	client      Client
	moduleInfos service.ModuleInfos
}

// attach records e as the currently-started extension instance. If a
// client has already been supplied via SetClient, it is forwarded to e's
// registry so that capabilities registered against e are advertised
// immediately. Any ModuleInfos captured by e from its host are cached so
// the OpAMP client owner can retrieve them via Registry.ModuleInfos.
func (m *instanceManager) attach(e *opampConnectionExtension) error {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.instance != nil {
		return fmt.Errorf(
			"only one opamp_connection extension may be configured per collector; %q is already configured",
			m.instance.id,
		)
	}
	m.instance = e
	m.moduleInfos = e.moduleInfos
	if m.client != nil {
		e.registry.setClient(m.client)
	}
	return nil
}

// detach clears the currently-started instance if it is e. The cached
// client is intentionally not cleared: it belongs to the opamp package,
// not to any one extension instance, and must be forwarded to the next
// instance that attaches.
func (m *instanceManager) detach(e *opampConnectionExtension) {
	m.mux.Lock()
	defer m.mux.Unlock()
	if m.instance == e {
		m.instance = nil
	}
}

// SetClient implements Registry.
func (m *instanceManager) SetClient(c Client) {
	m.mux.Lock()
	defer m.mux.Unlock()
	m.client = c
	if m.instance != nil {
		m.instance.registry.setClient(c)
	}
}

// ProcessMessage implements Registry.
func (m *instanceManager) ProcessMessage(cm *protobufs.CustomMessage) {
	m.mux.Lock()
	inst := m.instance
	m.mux.Unlock()
	if inst == nil {
		return
	}
	inst.registry.ProcessMessage(cm)
}

// ModuleInfos implements Registry.
func (m *instanceManager) ModuleInfos() service.ModuleInfos {
	m.mux.Lock()
	defer m.mux.Unlock()
	return m.moduleInfos
}

// opampConnectionExtension is the concrete extension implementation. It
// owns a per-instance customCapabilityRegistry whose lifetime matches the
// extension's: on Shutdown the registry — and every capability and
// message channel it holds — is abandoned. The package-level manager
// bridges whichever instance is currently attached to the long-lived
// OpAMP client.
type opampConnectionExtension struct {
	id          component.ID
	logger      *zap.Logger
	registry    *customCapabilityRegistry
	moduleInfos service.ModuleInfos
}

var (
	_ extension.Extension                          = (*opampConnectionExtension)(nil)
	_ opampcustommessages.CustomCapabilityRegistry = (*opampConnectionExtension)(nil)
)

func newExtension(id component.ID, logger *zap.Logger) *opampConnectionExtension {
	return &opampConnectionExtension{
		id:       id,
		logger:   logger,
		registry: newCustomCapabilityRegistry(logger),
	}
}

// Start attaches this extension instance to the package-level manager.
// Only a single opamp_connection extension may be configured per
// collector, since they would otherwise overwrite each other's advertised
// custom capabilities whenever a component registers or unregisters one.
// If host implements the hostcapabilities.ModuleInfo interface, the module
// info is captured here and cached on the manager so the OpAMP client
// owner can use it to build an AvailableComponents report.
func (e *opampConnectionExtension) Start(_ context.Context, host component.Host) error {
	if mi, ok := host.(hostcapabilities.ModuleInfo); ok {
		e.moduleInfos = mi.GetModuleInfos()
	}
	return manager.attach(e)
}

// Shutdown detaches this extension instance from the package-level
// manager. The per-instance registry is abandoned along with any
// capabilities registered against it.
func (e *opampConnectionExtension) Shutdown(_ context.Context) error {
	manager.detach(e)
	return nil
}

// Register implements opampcustommessages.CustomCapabilityRegistry.
func (e *opampConnectionExtension) Register(capability string, opts ...opampcustommessages.CustomCapabilityRegisterOption) (opampcustommessages.CustomCapabilityHandler, error) {
	return e.registry.Register(capability, opts...)
}
