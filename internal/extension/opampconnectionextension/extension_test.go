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
	"testing"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

// resetManager clears the package-level manager state between tests so a
// leaked instance or client reference from one test does not affect
// another.
func resetManager(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		manager.mux.Lock()
		manager.instance = nil
		manager.client = nil
		manager.mux.Unlock()
	})
}

func TestExtension_Start_RejectsSecondInstance(t *testing.T) {
	resetManager(t)

	ext1 := newExtension(component.MustNewIDWithName("opamp_connection", "one"), zap.NewNop())
	ext2 := newExtension(component.MustNewIDWithName("opamp_connection", "two"), zap.NewNop())

	require.NoError(t, ext1.Start(context.Background(), nil))
	t.Cleanup(func() { _ = ext1.Shutdown(context.Background()) })

	err := ext2.Start(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "only one opamp_connection extension")
}

func TestExtension_Shutdown_AllowsSubsequentStart(t *testing.T) {
	resetManager(t)

	ext1 := newExtension(component.MustNewIDWithName("opamp_connection", "first"), zap.NewNop())
	ext2 := newExtension(component.MustNewIDWithName("opamp_connection", "second"), zap.NewNop())

	require.NoError(t, ext1.Start(context.Background(), nil))
	require.NoError(t, ext1.Shutdown(context.Background()))

	// After the first extension shuts down, a second one can start in its
	// place (e.g. across a collector restart).
	require.NoError(t, ext2.Start(context.Background(), nil))
	t.Cleanup(func() { _ = ext2.Shutdown(context.Background()) })
}

func TestRegistry_ForwardsClientToCurrentInstance(t *testing.T) {
	resetManager(t)

	var advertised []*protobufs.CustomCapabilities
	client := mockCustomCapabilityClient{
		setCustomCapabilities: func(cc *protobufs.CustomCapabilities) error {
			advertised = append(advertised, cc)
			return nil
		},
	}

	ext := newExtension(component.MustNewIDWithName("opamp_connection", "x"), zap.NewNop())
	require.NoError(t, ext.Start(context.Background(), nil))
	t.Cleanup(func() { _ = ext.Shutdown(context.Background()) })

	// SetClient after Start forwards to the attached instance immediately.
	GetRegistry().SetClient(client)

	_, err := ext.Register("io.opentelemetry.teapot")
	require.NoError(t, err)
	require.Len(t, advertised, 1)
	require.Equal(t, []string{"io.opentelemetry.teapot"}, advertised[0].Capabilities)
}

func TestRegistry_ForwardsCachedClientToLateInstance(t *testing.T) {
	resetManager(t)

	var advertised []*protobufs.CustomCapabilities
	client := mockCustomCapabilityClient{
		setCustomCapabilities: func(cc *protobufs.CustomCapabilities) error {
			advertised = append(advertised, cc)
			return nil
		},
	}

	// Client supplied before any extension instance has started.
	GetRegistry().SetClient(client)

	ext := newExtension(component.MustNewIDWithName("opamp_connection", "late"), zap.NewNop())

	// Register before Start — the per-instance registry has no client
	// yet, so no advertisement goes out.
	_, err := ext.Register("io.opentelemetry.teapot")
	require.NoError(t, err)
	require.Empty(t, advertised)

	// Start forwards the cached client to the instance's registry, which
	// flushes accumulated capabilities.
	require.NoError(t, ext.Start(context.Background(), nil))
	t.Cleanup(func() { _ = ext.Shutdown(context.Background()) })

	require.Len(t, advertised, 1)
	require.Equal(t, []string{"io.opentelemetry.teapot"}, advertised[0].Capabilities)
}

func TestRegistry_CachesClientAcrossInstances(t *testing.T) {
	resetManager(t)

	var advertised []*protobufs.CustomCapabilities
	client := mockCustomCapabilityClient{
		setCustomCapabilities: func(cc *protobufs.CustomCapabilities) error {
			advertised = append(advertised, cc)
			return nil
		},
	}

	// First instance starts, client is wired, then it shuts down. This
	// mirrors the collector lifecycle before a config reload.
	ext1 := newExtension(component.MustNewIDWithName("opamp_connection", "first"), zap.NewNop())
	require.NoError(t, ext1.Start(context.Background(), nil))
	GetRegistry().SetClient(client)
	require.NoError(t, ext1.Shutdown(context.Background()))

	// A second instance starts after the restart. No second SetClient
	// call is made — the manager's cached client must reach the new
	// registry automatically.
	ext2 := newExtension(component.MustNewIDWithName("opamp_connection", "second"), zap.NewNop())
	require.NoError(t, ext2.Start(context.Background(), nil))
	t.Cleanup(func() { _ = ext2.Shutdown(context.Background()) })

	advertised = nil
	_, err := ext2.Register("io.opentelemetry.teapot")
	require.NoError(t, err)
	require.Len(t, advertised, 1, "capability registered after restart should be advertised")
	require.Equal(t, []string{"io.opentelemetry.teapot"}, advertised[0].Capabilities)
}

func TestRegistry_ProcessMessageRoutesToCurrentInstance(t *testing.T) {
	resetManager(t)

	ext := newExtension(component.MustNewIDWithName("opamp_connection", "x"), zap.NewNop())
	require.NoError(t, ext.Start(context.Background(), nil))
	t.Cleanup(func() { _ = ext.Shutdown(context.Background()) })
	GetRegistry().SetClient(mockCustomCapabilityClient{})

	handler, err := ext.Register("io.opentelemetry.teapot")
	require.NoError(t, err)

	msg := &protobufs.CustomMessage{
		Capability: "io.opentelemetry.teapot",
		Type:       "steep",
		Data:       []byte("blackTea"),
	}
	GetRegistry().ProcessMessage(msg)
	require.Equal(t, msg, <-handler.Message())
}

func TestRegistry_ProcessMessageWithoutInstanceIsNoop(t *testing.T) {
	resetManager(t)

	// No extension attached — should be a harmless no-op.
	GetRegistry().ProcessMessage(&protobufs.CustomMessage{
		Capability: "io.opentelemetry.teapot",
		Type:       "steep",
	})
}
