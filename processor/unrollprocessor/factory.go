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

package unrollprocessor // import "github.com/observiq/bindplane-otel-collector/processor/unrollprocessor"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

var processorCapabilities = consumer.Capabilities{MutatesData: true}

// componentType is the value of the "type" key in configuration.
var componentType = component.MustNewType("unroll")

const (
	stability = component.StabilityLevelAlpha
)

// NewFactory returns a new factory for the Transform processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		componentType,
		createDefaultConfig,
		processor.WithLogs(createLogsProcessor, stability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		Field: UnrollFieldBody,
	}
}

func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	oCfg := cfg.(*Config)

	proc, err := newUnrollProcessor(oCfg)
	if err != nil {
		return nil, fmt.Errorf("invalid config for \"unroll\" processor %w", err)
	}
	return processorhelper.NewLogs(
		ctx,
		set,
		cfg,
		nextConsumer,
		proc.ProcessLogs,
		processorhelper.WithCapabilities(processorCapabilities))
}
