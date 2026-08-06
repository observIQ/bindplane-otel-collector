// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package observiq

import (
	"context"

	"github.com/open-telemetry/opamp-go/client"
	"github.com/open-telemetry/opamp-go/protobufs"
)

// gatedOpAMPClient wraps a client.OpAMPClient so that every call which
// results in a message being sent to the Server first waits on a sendGate.
// Start, Stop, and AgentDescription are passed straight through since they
// don't send new outgoing data (or, in the case of Stop, must never be
// delayed).
type gatedOpAMPClient struct {
	client.OpAMPClient
	gate *sendGate
}

// newGatedOpAMPClient wraps underlying so that outgoing messages honor gate.
func newGatedOpAMPClient(underlying client.OpAMPClient, gate *sendGate) client.OpAMPClient {
	return &gatedOpAMPClient{OpAMPClient: underlying, gate: gate}
}

func (g *gatedOpAMPClient) SetAgentDescription(descr *protobufs.AgentDescription) error {
	g.gate.wait()
	return g.OpAMPClient.SetAgentDescription(descr)
}

func (g *gatedOpAMPClient) SetHealth(health *protobufs.ComponentHealth) error {
	g.gate.wait()
	return g.OpAMPClient.SetHealth(health)
}

func (g *gatedOpAMPClient) UpdateEffectiveConfig(ctx context.Context) error {
	g.gate.wait()
	return g.OpAMPClient.UpdateEffectiveConfig(ctx)
}

func (g *gatedOpAMPClient) SetRemoteConfigStatus(status *protobufs.RemoteConfigStatus) error {
	g.gate.wait()
	return g.OpAMPClient.SetRemoteConfigStatus(status)
}

func (g *gatedOpAMPClient) SetPackageStatuses(statuses *protobufs.PackageStatuses) error {
	g.gate.wait()
	return g.OpAMPClient.SetPackageStatuses(statuses)
}

func (g *gatedOpAMPClient) RequestConnectionSettings(request *protobufs.ConnectionSettingsRequest) error {
	g.gate.wait()
	return g.OpAMPClient.RequestConnectionSettings(request)
}

func (g *gatedOpAMPClient) SetCustomCapabilities(customCapabilities *protobufs.CustomCapabilities) error {
	g.gate.wait()
	return g.OpAMPClient.SetCustomCapabilities(customCapabilities)
}

func (g *gatedOpAMPClient) SetFlags(flags protobufs.AgentToServerFlags) {
	g.gate.wait()
	g.OpAMPClient.SetFlags(flags)
}

func (g *gatedOpAMPClient) SendCustomMessage(message *protobufs.CustomMessage) (chan struct{}, error) {
	g.gate.wait()
	return g.OpAMPClient.SendCustomMessage(message)
}

func (g *gatedOpAMPClient) SetAvailableComponents(components *protobufs.AvailableComponents) error {
	g.gate.wait()
	return g.OpAMPClient.SetAvailableComponents(components)
}

func (g *gatedOpAMPClient) SetCapabilities(capabilities *protobufs.AgentCapabilities) error {
	g.gate.wait()
	return g.OpAMPClient.SetCapabilities(capabilities)
}
