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
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configunmarshaler"
	"gopkg.in/yaml.v2"
)

// ConfigProvider implements the service.ConfigProvider interface
type ConfigProvider struct {
	components   *ComponentMap
	errChan      chan error
	unmarshaller configunmarshaler.ConfigUnmarshaler
}

// createConfigProvider creates a config provider
func createConfigProvider(components *ComponentMap) *ConfigProvider {
	return &ConfigProvider{
		components:   components,
		errChan:      make(chan error),
		unmarshaller: configunmarshaler.NewDefault(),
	}
}

// Get returns the underlying config of the provider
func (c *ConfigProvider) Get(_ context.Context, factories component.Factories) (*config.Config, error) {
	configMap := c.components.ToConfigMap()
	config, err := c.unmarshaller.Unmarshal(configMap, factories)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config map: %w", err)
	}

	return config, nil
}

// Watch returns a channel that indicates updates to the config
func (c *ConfigProvider) Watch() <-chan error {
	return c.errChan
}

// Shutdown always returns nil and is a no-op
func (c *ConfigProvider) Shutdown(_ context.Context) error {
	return nil
}

// GetRequiredFactories returns the factories required for the configured components
func (c *ConfigProvider) GetRequiredFactories(host component.Host, emitterFactory component.ExporterFactory) (*component.Factories, error) {
	receiverFactories, err := c.getReceiverFactories(host)
	if err != nil {
		return nil, fmt.Errorf("failed to get receiver factories: %w", err)
	}

	processorFactories, err := c.getProcessorFactories(host)
	if err != nil {
		return nil, fmt.Errorf("failed to get processor factories: %w", err)
	}

	extensionFactories, err := c.getExtensionFactories(host)
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

// getReceiverFactories returns the receiver factories required for the configured receivers
func (c *ConfigProvider) getReceiverFactories(host component.Host) (map[config.Type]component.ReceiverFactory, error) {
	factories := map[config.Type]component.ReceiverFactory{}
	for receiver := range c.components.Receivers {
		receiverType := parseComponentType(receiver)
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

// getProcessorFactories returns the processor factories required for the configured processors
func (c *ConfigProvider) getProcessorFactories(host component.Host) (map[config.Type]component.ProcessorFactory, error) {
	factories := map[config.Type]component.ProcessorFactory{}
	for processor := range c.components.Processors {
		processorType := parseComponentType(processor)
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

// getExtensionFactories returns the extension factories required for the configured extensions
func (c *ConfigProvider) getExtensionFactories(host component.Host) (map[config.Type]component.ExtensionFactory, error) {
	factories := map[config.Type]component.ExtensionFactory{}
	for extension := range c.components.Extensions {
		extensionType := parseComponentType(extension)
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

// unmarshalComponents unmarshals a component map from yaml
func unmarshalComponentMap(bytes []byte) (*ComponentMap, error) {
	var components ComponentMap
	if err := yaml.Unmarshal(bytes, &components); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	components.Exporters = map[string]interface{}{
		emitterTypeStr: nil,
	}

	for key, pipeline := range components.Service.Pipelines {
		pipeline.Exporters = []string{emitterTypeStr}
		components.Service.Pipelines[key] = pipeline
	}

	return &components, nil
}

// ComponentMap is a map of configured open telemetry components
type ComponentMap struct {
	Receivers  map[string]interface{} `yaml:"receivers,omitempty"`
	Processors map[string]interface{} `yaml:"processors,omitempty"`
	Exporters  map[string]interface{} `yaml:"exporters,omitempty"`
	Extensions map[string]interface{} `yaml:"extensions,omitempty"`
	Service    ServiceMap             `yaml:"service,omitempty"`
}

// ServiceMap is a map of service components
type ServiceMap struct {
	Extensions []string               `yaml:"extensions,omitempty"`
	Pipelines  map[string]PipelineMap `yaml:"pipelines,omitempty"`
}

// PipelineMap is a map of pipeline components
type PipelineMap struct {
	Receivers  []string `yaml:"receivers,omitempty"`
	Processors []string `yaml:"processors,omitempty"`
	Exporters  []string `yaml:"exporters,omitempty"`
}

// ToConfigMap returns the component map as a config map
func (c *ComponentMap) ToConfigMap() *config.Map {
	pipelines := map[string]interface{}{}
	for key, pipeline := range c.Service.Pipelines {
		pipelineMap := map[string]interface{}{}
		pipelineMap["receivers"] = pipeline.Receivers
		pipelineMap["processors"] = pipeline.Processors
		pipelineMap["exporters"] = pipeline.Exporters
		pipelines[key] = pipelineMap
	}

	stringMap := map[string]interface{}{
		"receivers":  c.Receivers,
		"processors": c.Processors,
		"exporters":  c.Exporters,
		"extensions": c.Extensions,
		"service": map[string]interface{}{
			"extensions": c.Service.Extensions,
			"pipelines":  pipelines,
		},
	}

	return config.NewMapFromStringMap(stringMap)
}
