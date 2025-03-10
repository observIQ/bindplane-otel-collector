// Copyright  Bindplane, Inc.
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

package observiq

import (
	"os"
	"runtime"

	ios "github.com/observiq/bindplane-otel-collector/internal/os"
	"github.com/observiq/bindplane-otel-collector/opamp"
	"github.com/open-telemetry/opamp-go/protobufs"
	"go.uber.org/zap"
)

// identity contains identifying information about the Collector
type identity struct {
	agentID     opamp.AgentID
	agentName   *string
	serviceName string
	version     string
	labels      *string
	oSArch      string
	oSDetails   string
	oSFamily    string
	hostname    string
	mac         string
}

// newIdentity constructs a new identity for this collector
func newIdentity(logger *zap.Logger, config opamp.Config, version string) *identity {
	// Grab various fields from OS
	hostname, err := ios.Hostname()
	if err != nil {
		logger.Warn("Failed to retrieve hostname for collector. Creating partial identity", zap.Error(err))
	}

	name, err := ios.Name()
	if err != nil {
		logger.Warn("Failed to retrieve host details on collector. Creating partial identity", zap.Error(err))
	}

	return &identity{
		agentID:     config.AgentID,
		agentName:   config.AgentName,
		serviceName: "com.observiq.collector", // Hardcoded defines this type of agent to the server
		version:     version,
		labels:      config.Labels,
		oSArch:      runtime.GOARCH,
		oSDetails:   name,
		oSFamily:    runtime.GOOS,
		hostname:    hostname,
		mac:         ios.MACAddress(),
	}
}

// Copy creates a deep copy of this identity
func (i identity) Copy() *identity {
	identCpy := &identity{
		agentID:     i.agentID,
		serviceName: i.serviceName,
		version:     i.version,
		oSArch:      i.oSArch,
		oSDetails:   i.oSDetails,
		oSFamily:    i.oSFamily,
		hostname:    i.hostname,
		mac:         i.mac,
	}

	if i.agentName != nil {
		identCpy.agentName = new(string)
		*identCpy.agentName = *i.agentName
	}

	if i.labels != nil {
		identCpy.labels = new(string)
		*identCpy.labels = *i.labels
	}

	return identCpy
}

func (i *identity) ToAgentDescription() *protobufs.AgentDescription {
	identifyingAttributes := []*protobufs.KeyValue{
		opamp.StringKeyValue("service.instance.id", i.agentID.String()),
		opamp.StringKeyValue("service.name", i.serviceName),
		opamp.StringKeyValue("service.version", i.version),
	}

	if i.agentName != nil {
		identifyingAttributes = append(identifyingAttributes, opamp.StringKeyValue("service.instance.name", *i.agentName))
	} else {
		identifyingAttributes = append(identifyingAttributes, opamp.StringKeyValue("service.instance.name", i.hostname))
	}

	nonIdentifyingAttributes := []*protobufs.KeyValue{
		opamp.StringKeyValue("os.arch", i.oSArch),
		opamp.StringKeyValue("os.details", i.oSDetails),
		opamp.StringKeyValue("os.family", i.oSFamily),
		opamp.StringKeyValue("host.name", i.hostname),
		opamp.StringKeyValue("host.mac_address", i.mac),
	}

	if i.labels != nil {
		nonIdentifyingAttributes = append(nonIdentifyingAttributes, opamp.StringKeyValue("service.labels", *i.labels))
	}

	key := os.Getenv("OTEL_AES_CREDENTIAL_PROVIDER")
	if key != "" {
		nonIdentifyingAttributes = append(nonIdentifyingAttributes, opamp.StringKeyValue("service.key", key))
	}

	agentDesc := &protobufs.AgentDescription{
		IdentifyingAttributes:    identifyingAttributes,
		NonIdentifyingAttributes: nonIdentifyingAttributes,
	}

	return agentDesc
}
