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

// This file is ported from the upstream opampextension at
//
//	github.com/open-telemetry/opentelemetry-collector-contrib/extension/opampextension/opamp_agent.go @ v0.151.0
//
// The four AvailableComponents helpers (initAvailableComponents,
// generateAvailableComponentsHash, addComponentTypeComponentsToStringBuilder,
// createComponentTypeAvailableComponentDetails) are kept here with the same
// names so the upstream source can be diffed and bug fixes ported by hand.
//
// generateAvailableComponentsHash, addComponentTypeComponentsToStringBuilder,
// and createComponentTypeAvailableComponentDetails are byte-identical to
// upstream. initAvailableComponents is restructured from a *opampAgent method
// to a standalone function returning the *protobufs.AvailableComponents value
// (this distribution's OpAMP client struct differs from upstream's), but the
// body is otherwise unchanged.
//
// When updating, diff against the upstream file at the pinned version above.

package observiq

import (
	"crypto/sha256"
	"sort"
	"strings"

	"github.com/open-telemetry/opamp-go/protobufs"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service"
)

// initAvailableComponents builds the AvailableComponents proto from a
// service.ModuleInfos. Returns the populated value; the caller is
// responsible for skipping SetAvailableComponents when the
// ReportsAvailableComponents capability is not advertised.
func initAvailableComponents(moduleInfos service.ModuleInfos) *protobufs.AvailableComponents {
	return &protobufs.AvailableComponents{
		Hash: generateAvailableComponentsHash(moduleInfos),
		Components: map[string]*protobufs.ComponentDetails{
			"receivers": {
				SubComponentMap: createComponentTypeAvailableComponentDetails(moduleInfos.Receiver),
			},
			"processors": {
				SubComponentMap: createComponentTypeAvailableComponentDetails(moduleInfos.Processor),
			},
			"exporters": {
				SubComponentMap: createComponentTypeAvailableComponentDetails(moduleInfos.Exporter),
			},
			"extensions": {
				SubComponentMap: createComponentTypeAvailableComponentDetails(moduleInfos.Extension),
			},
			"connectors": {
				SubComponentMap: createComponentTypeAvailableComponentDetails(moduleInfos.Connector),
			},
		},
	}
}

func generateAvailableComponentsHash(moduleInfos service.ModuleInfos) []byte {
	var builder strings.Builder

	addComponentTypeComponentsToStringBuilder(&builder, moduleInfos.Receiver, "receiver")
	addComponentTypeComponentsToStringBuilder(&builder, moduleInfos.Processor, "processor")
	addComponentTypeComponentsToStringBuilder(&builder, moduleInfos.Exporter, "exporter")
	addComponentTypeComponentsToStringBuilder(&builder, moduleInfos.Extension, "extension")
	addComponentTypeComponentsToStringBuilder(&builder, moduleInfos.Connector, "connector")

	// Compute the SHA-256 hash of the serialized representation.
	hash := sha256.Sum256([]byte(builder.String()))
	return hash[:]
}

func addComponentTypeComponentsToStringBuilder(builder *strings.Builder, componentTypeComponents map[component.Type]service.ModuleInfo, componentType string) {
	// Collect components and sort them to ensure deterministic ordering.
	components := make([]component.Type, 0, len(componentTypeComponents))
	for k := range componentTypeComponents {
		components = append(components, k)
	}
	sort.Slice(components, func(i, j int) bool {
		return components[i].String() < components[j].String()
	})

	// Append the component type and its sorted key-value pairs.
	builder.WriteString(componentType + ":")
	for _, k := range components {
		builder.WriteString(k.String() + "=" + componentTypeComponents[k].BuilderRef + ";")
	}
}

func createComponentTypeAvailableComponentDetails(componentTypeComponents map[component.Type]service.ModuleInfo) map[string]*protobufs.ComponentDetails {
	availableComponentDetails := map[string]*protobufs.ComponentDetails{}
	for componentType, r := range componentTypeComponents {
		availableComponentDetails[componentType.String()] = &protobufs.ComponentDetails{
			Metadata: []*protobufs.KeyValue{
				{
					Key: "code.namespace",
					Value: &protobufs.AnyValue{
						Value: &protobufs.AnyValue_StringValue{
							StringValue: r.BuilderRef,
						},
					},
				},
			},
		}
	}
	return availableComponentDetails
}
