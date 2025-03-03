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

// Package factories provides factories for components in the collector
package factories

import (
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/multierr"
)

// DefaultFactories returns the default factories used by the Bindplane Agent
func DefaultFactories() (otelcol.Factories, error) {
	return combineFactories(defaultReceivers, defaultProcessors, defaultExporters, defaultExtensions, defaultConnectors)
}

// combineFactories combines the supplied factories into a single Factories struct.
// Any errors encountered will also be combined into a single error.
func combineFactories(receivers []receiver.Factory, processors []processor.Factory,
	exporters []exporter.Factory, extensions []extension.Factory,
	connectors []connector.Factory) (otelcol.Factories, error) {
	var errs []error

	receiverMap, err := receiver.MakeFactoryMap(receivers...)
	if err != nil {
		errs = append(errs, err)
	}

	processorMap, err := processor.MakeFactoryMap(processors...)
	if err != nil {
		errs = append(errs, err)
	}

	exporterMap, err := exporter.MakeFactoryMap(exporters...)
	if err != nil {
		errs = append(errs, err)
	}

	extensionMap, err := extension.MakeFactoryMap(extensions...)
	if err != nil {
		errs = append(errs, err)
	}

	connectorMap, err := connector.MakeFactoryMap(connectors...)
	if err != nil {
		errs = append(errs, err)
	}

	return otelcol.Factories{
		Receivers:  receiverMap,
		Processors: processorMap,
		Exporters:  exporterMap,
		Extensions: extensionMap,
		Connectors: connectorMap,
	}, multierr.Combine(errs...)
}
