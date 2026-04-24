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

// This file mirrors the AvailableComponents implementation in
// opamp_agent.go in the opampextension package of
// opentelemetry-collector-contrib
// (github.com/open-telemetry/opentelemetry-collector-contrib/extension/opampextension).

package opampconnectionextension

import (
	"crypto/sha256"
	"sort"
	"strings"

	"github.com/open-telemetry/opamp-go/protobufs"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/hostcapabilities"
)

// buildAvailableComponents returns the set of components configured in this
// collector. If the supplied host does not implement
// hostcapabilities.ComponentFactory, deprecated-alias entries are omitted.
func buildAvailableComponents(host component.Host, moduleInfos service.ModuleInfos) *protobufs.AvailableComponents {
	var factorySource hostcapabilities.ComponentFactory
	if cf, ok := host.(hostcapabilities.ComponentFactory); ok {
		factorySource = cf
	}

	return &protobufs.AvailableComponents{
		Hash: generateAvailableComponentsHash(moduleInfos),
		Components: map[string]*protobufs.ComponentDetails{
			"receivers": {
				SubComponentMap: createComponentTypeAvailableComponentDetails(factorySource, component.KindReceiver, moduleInfos.Receiver),
			},
			"processors": {
				SubComponentMap: createComponentTypeAvailableComponentDetails(factorySource, component.KindProcessor, moduleInfos.Processor),
			},
			"exporters": {
				SubComponentMap: createComponentTypeAvailableComponentDetails(factorySource, component.KindExporter, moduleInfos.Exporter),
			},
			"extensions": {
				SubComponentMap: createComponentTypeAvailableComponentDetails(factorySource, component.KindExtension, moduleInfos.Extension),
			},
			"connectors": {
				SubComponentMap: createComponentTypeAvailableComponentDetails(factorySource, component.KindConnector, moduleInfos.Connector),
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

	hash := sha256.Sum256([]byte(builder.String()))
	return hash[:]
}

func addComponentTypeComponentsToStringBuilder(builder *strings.Builder, componentTypeComponents map[component.Type]service.ModuleInfo, componentType string) {
	components := make([]component.Type, 0, len(componentTypeComponents))
	for k := range componentTypeComponents {
		components = append(components, k)
	}
	sort.Slice(components, func(i, j int) bool {
		return components[i].String() < components[j].String()
	})

	builder.WriteString(componentType + ":")
	for _, k := range components {
		builder.WriteString(k.String() + "=" + componentTypeComponents[k].BuilderRef + ";")
	}
}

// typeAliasHolder mirrors the DeprecatedAlias accessor implemented by
// factories built via xreceiver/xprocessor/xexporter/xextension/xconnector
// NewFactory. The collector's interface lives in an internal package, so we
// restate it here and rely on Go's structural typing.
type typeAliasHolder interface {
	DeprecatedAlias() component.Type
}

func createComponentTypeAvailableComponentDetails(factorySource hostcapabilities.ComponentFactory, kind component.Kind, componentTypeComponents map[component.Type]service.ModuleInfo) map[string]*protobufs.ComponentDetails {
	availableComponentDetails := map[string]*protobufs.ComponentDetails{}
	for componentType, r := range componentTypeComponents {
		details := &protobufs.ComponentDetails{
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
		availableComponentDetails[componentType.String()] = details

		if factorySource == nil {
			continue
		}
		factory := factorySource.GetFactory(kind, componentType)
		if factory == nil {
			continue
		}
		holder, ok := factory.(typeAliasHolder)
		if !ok {
			continue
		}
		alias := holder.DeprecatedAlias()
		if alias.String() == "" {
			continue
		}
		availableComponentDetails[alias.String()] = details
	}
	return availableComponentDetails
}
