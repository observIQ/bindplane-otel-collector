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

package ocsfstandardizationprocessor

import (
	"context"
	"errors"

	"github.com/observiq/bindplane-otel-collector/processor/ocsfstandardizationprocessor/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

var (
	// errInvalidConfigType is returned when the configuration is not of the expected type.
	errInvalidConfigType = errors.New("config is not of type ocsfstandardizationprocessor.Config")
)

const (
	stability = component.StabilityLevelAlpha
)

var (
	consumerCapabilities = consumer.Capabilities{MutatesData: true}
)

// NewFactory creates a new ProcessorFactory with default configuration
func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		processor.WithLogs(createLogsProcessor, stability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	oCfg, ok := cfg.(*Config)
	if !ok {
		return nil, errInvalidConfigType
	}
	processor, err := newOCSFStandardizationProcessor(set.Logger, oCfg)
	if err != nil {
		return nil, err
	}

	return processorhelper.NewLogs(
		ctx,
		set,
		cfg,
		nextConsumer,
		processor.processLogs,
		processorhelper.WithCapabilities(consumerCapabilities),
	)
}
