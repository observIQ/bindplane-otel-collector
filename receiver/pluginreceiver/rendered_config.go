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
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/converter/expandconverter"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/service"
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
		emitterTypeStr: nil,
	}

	for key, pipeline := range renderedCfg.Service.Pipelines {
		pipeline.Exporters = []string{emitterTypeStr}
		renderedCfg.Service.Pipelines[key] = pipeline
	}

	// Hardcode telemetry to none so the collector setup for the plugin doesn't record metrics
	renderedCfg.Service.Telemetry.Metrics.Level = "none"

	return &renderedCfg, nil
}

// GetConfigProvider returns a config provider for the rendered config
func (r *RenderedConfig) GetConfigProvider() (service.ConfigProvider, error) {
	bytes, err := yaml.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config as bytes: %w", err)
	}

	location := fmt.Sprintf("yaml:%s", bytes)
	provider := yamlprovider.New()
	converter := expandconverter.New()
	settings := service.ConfigProviderSettings{
		ResolverSettings: confmap.ResolverSettings{
			URIs:       []string{location},
			Providers:  map[string]confmap.Provider{provider.Scheme(): provider},
			Converters: []confmap.Converter{converter},
		},
	}

	return service.NewConfigProvider(settings)
}

// GetRequiredFactories finds and returns the factories required for the rendered config
func (r *RenderedConfig) GetRequiredFactories(host component.Host, emitterFactory component.ExporterFactory) (*component.Factories, error) {
	receiverFactories, err := r.getReceiverFactories(host)
	if err != nil {
		return nil, fmt.Errorf("failed to get receiver factories: %w", err)
	}

	processorFactories, err := r.getProcessorFactories(host)
	if err != nil {
		return nil, fmt.Errorf("failed to get processor factories: %w", err)
	}

	extensionFactories, err := r.getExtensionFactories(host)
	if err != nil {
		return nil, fmt.Errorf("failed to get extension factories: %w", err)
	}

	exporterFactories := map[config.Type]component.ExporterFactory{
		emitterFactory.Type(): emitterFactory,
	}

	return &component.Factories{
		Receivers:  receiverFactories,
		Processors: processorFactories,
		Exporters:  exporterFactories,
		Extensions: extensionFactories,
	}, nil
}

// getReceiverFactories returns the receiver factories required for the rendered config
func (r *RenderedConfig) getReceiverFactories(host component.Host) (map[config.Type]component.ReceiverFactory, error) {
	factories := map[config.Type]component.ReceiverFactory{}
	for receiverID := range r.Receivers {
		receiverType := parseComponentType(receiverID)
		if _, ok := factories[receiverType]; ok {
			continue
		}

		factory := host.GetFactory(component.KindReceiver, receiverType)
		receiverFactory, ok := factory.(component.ReceiverFactory)
		if !ok {
			return nil, fmt.Errorf("receiver factory %s is missing from host", receiverType)
		}

		factories[receiverType] = receiverFactory
	}

	return factories, nil
}

// getProcessorFactories returns the processor factories required for the rendered config
func (r *RenderedConfig) getProcessorFactories(host component.Host) (map[config.Type]component.ProcessorFactory, error) {
	factories := map[config.Type]component.ProcessorFactory{}
	for processorID := range r.Processors {
		processorType := parseComponentType(processorID)
		if _, ok := factories[processorType]; ok {
			continue
		}

		factory := host.GetFactory(component.KindProcessor, processorType)
		processorFactory, ok := factory.(component.ProcessorFactory)
		if !ok {
			return nil, fmt.Errorf("processor factory %s is missing from host", processorType)
		}

		factories[processorType] = processorFactory
	}

	return factories, nil
}

// getExtensionFactories returns the extension factories required for the rendered config
func (r *RenderedConfig) getExtensionFactories(host component.Host) (map[config.Type]component.ExtensionFactory, error) {
	factories := map[config.Type]component.ExtensionFactory{}
	for extensionID := range r.Extensions {
		extensionType := parseComponentType(extensionID)
		if _, ok := factories[extensionType]; ok {
			continue
		}

		factory := host.GetFactory(component.KindExtension, extensionType)
		extensionFactory, ok := factory.(component.ExtensionFactory)
		if !ok {
			return nil, fmt.Errorf("extension factory %s is missing from host", extensionType)
		}

		factories[extensionType] = extensionFactory
	}

	return factories, nil
}

// parseComponentType parses a component type from a string
func parseComponentType(value string) config.Type {
	componentID, _ := config.NewComponentIDFromString(value)
	return componentID.Type()
}
