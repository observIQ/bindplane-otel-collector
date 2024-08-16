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

package pluginreceiver

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpsprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"gopkg.in/yaml.v2"
)

// RenderedConfig is the rendered config of a plugin
type RenderedConfig struct {
	Receivers  map[string]any `yaml:"receivers,omitempty"`
	Processors map[string]any `yaml:"processors,omitempty"`
	Exporters  map[string]any `yaml:"exporters,omitempty"`
	Extensions map[string]any `yaml:"extensions,omitempty"`
	Service    ServiceConfig  `yaml:"service,omitempty"`
}

// ServiceConfig is the config of a collector service
type ServiceConfig struct {
	Extensions []string                  `yaml:"extensions,omitempty"`
	Pipelines  map[string]PipelineConfig `yaml:"pipelines,omitempty"`
	Telemetry  TelemetryConfig           `yaml:"telemetry,omitempty"`
}

// PipelineConfig is the config of a pipeline
type PipelineConfig struct {
	Receivers  []string `yaml:"receivers,omitempty"`
	Processors []string `yaml:"processors,omitempty"`
	Exporters  []string `yaml:"exporters,omitempty"`
}

// TelemetryConfig is a representation of collector telemetry settings
type TelemetryConfig struct {
	Metrics MetricsConfig `yaml:"metrics,omitempty"`
}

// MetricsConfig exposes the level of the telemetry metrics
type MetricsConfig struct {
	Level string `yaml:"level,omitempty"`
}

// NewRenderedConfig creates a RenderedConfig with statically overwritten Exporters info
func NewRenderedConfig(yamlBytes []byte) (*RenderedConfig, error) {
	var renderedCfg RenderedConfig
	if err := yaml.Unmarshal(yamlBytes, &renderedCfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml bytes: %w", err)
	}

	renderedCfg.Exporters = map[string]any{
		emitterType.String(): nil,
	}

	for key, pipeline := range renderedCfg.Service.Pipelines {
		pipeline.Exporters = []string{emitterType.String()}
		renderedCfg.Service.Pipelines[key] = pipeline
	}

	// Hardcode telemetry to none so the collector setup for the plugin doesn't record metrics
	renderedCfg.Service.Telemetry.Metrics.Level = "none"

	return &renderedCfg, nil
}

// GetConfigProviderSettings returns config provider settings for the rendered config
func (r *RenderedConfig) GetConfigProviderSettings() (*otelcol.ConfigProviderSettings, error) {
	bytes, err := yaml.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config as bytes: %w", err)
	}

	location := fmt.Sprintf("yaml:%s", bytes)
	settings := otelcol.ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs: []string{location},
			ProviderFactories: []confmap.ProviderFactory{
				fileprovider.NewFactory(),
				yamlprovider.NewFactory(),
				httpsprovider.NewFactory(),
				envprovider.NewFactory(),
			},
			ConverterFactories: []confmap.ConverterFactory{},
			DefaultScheme:      "env",
		},
	}

	return &settings, nil
}

// host is an interface that the component.Host passed to GetRequiredFactories must implement
type host interface {
	component.Host
	GetFactory(component.Kind, component.Type) component.Factory
}

// GetRequiredFactories finds and returns the factories required for the rendered config
func (r *RenderedConfig) GetRequiredFactories(h component.Host, emitterFactory exporter.Factory) (*otelcol.Factories, error) {
	rcHost, ok := h.(host)
	if !ok {
		return nil, fmt.Errorf("the RenderedConfig is not compatible with the provided component.host")
	}

	receiverFactories, err := r.getReceiverFactories(rcHost)
	if err != nil {
		return nil, fmt.Errorf("failed to get receiver factories: %w", err)
	}

	processorFactories, err := r.getProcessorFactories(rcHost)
	if err != nil {
		return nil, fmt.Errorf("failed to get processor factories: %w", err)
	}

	extensionFactories, err := r.getExtensionFactories(rcHost)
	if err != nil {
		return nil, fmt.Errorf("failed to get extension factories: %w", err)
	}

	exporterFactories := map[component.Type]exporter.Factory{
		emitterFactory.Type(): emitterFactory,
	}

	return &otelcol.Factories{
		Receivers:  receiverFactories,
		Processors: processorFactories,
		Exporters:  exporterFactories,
		Extensions: extensionFactories,
	}, nil
}

// getReceiverFactories returns the receiver factories required for the rendered config
func (r *RenderedConfig) getReceiverFactories(host host) (map[component.Type]receiver.Factory, error) {
	factories := map[component.Type]receiver.Factory{}
	for receiverID := range r.Receivers {
		receiverType, err := parseComponentType(receiverID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse receiverID '%s': %w", receiverID, err)
		}
		if _, ok := factories[receiverType]; ok {
			continue
		}

		factory := host.GetFactory(component.KindReceiver, receiverType)
		receiverFactory, ok := factory.(receiver.Factory)
		if !ok {
			return nil, fmt.Errorf("receiver factory %s is missing from host", receiverType)
		}

		factories[receiverType] = receiverFactory
	}

	return factories, nil
}

// getProcessorFactories returns the processor factories required for the rendered config
func (r *RenderedConfig) getProcessorFactories(host host) (map[component.Type]processor.Factory, error) {
	factories := map[component.Type]processor.Factory{}
	for processorID := range r.Processors {
		processorType, err := parseComponentType(processorID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse processorID '%s': %w", processorID, err)
		}
		if _, ok := factories[processorType]; ok {
			continue
		}

		factory := host.GetFactory(component.KindProcessor, processorType)
		processorFactory, ok := factory.(processor.Factory)
		if !ok {
			return nil, fmt.Errorf("processor factory %s is missing from host", processorType)
		}

		factories[processorType] = processorFactory
	}

	return factories, nil
}

// getExtensionFactories returns the extension factories required for the rendered config
func (r *RenderedConfig) getExtensionFactories(host host) (map[component.Type]extension.Factory, error) {
	factories := map[component.Type]extension.Factory{}
	for extensionID := range r.Extensions {
		extensionType, err := parseComponentType(extensionID)
		if err != nil {
			return nil, fmt.Errorf("failed to parse extensionID '%s': %w", extensionID, err)
		}
		if _, ok := factories[extensionType]; ok {
			continue
		}

		factory := host.GetFactory(component.KindExtension, extensionType)
		extensionFactory, ok := factory.(extension.Factory)
		if !ok {
			return nil, fmt.Errorf("extension factory %s is missing from host", extensionType)
		}

		factories[extensionType] = extensionFactory
	}

	return factories, nil
}

// parseComponentType parses a component type from a string
func parseComponentType(value string) (component.Type, error) {
	id := component.ID{}
	if err := id.UnmarshalText([]byte(value)); err != nil {
		return component.Type{}, err
	}
	return id.Type(), nil
}
