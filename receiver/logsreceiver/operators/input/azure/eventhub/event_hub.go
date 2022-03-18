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

package eventhub

import (
	"context"

	"github.com/observiq/observiq-otel-collector/receiver/logsreceiver/operators/input/azure"
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/helper"
)

const operatorName = "azure_event_hub_input"

func init() {
	operator.Register(operatorName, func() operator.Builder { return NewEventHubConfig("") })
}

// NewEventHubConfig creates a new Azure Event Hub input config with default values
func NewEventHubConfig(operatorID string) *InputConfig {
	return &InputConfig{
		InputConfig: helper.NewInputConfig(operatorID, operatorName),
		Config: azure.Config{
			PrefetchCount: 1000,
			StartAt:       "end",
		},
	}
}

// InputConfig is the configuration of a Azure Event Hub input operator.
type InputConfig struct {
	helper.InputConfig `yaml:",inline"`
	azure.Config       `yaml:",inline"`
}

// Build will build a Azure Event Hub input operator.
func (c *InputConfig) Build(buildContext operator.BuildContext) ([]operator.Operator, error) {
	if err := c.Config.Build(buildContext, c.InputConfig); err != nil {
		return nil, err
	}

	eventHubInput := &Input{
		EventHub: azure.EventHub{
			Config: c.Config,
		},
	}
	return []operator.Operator{eventHubInput}, nil
}

// Input is an operator that reads input from Azure Event Hub.
type Input struct {
	azure.EventHub
}

// Start will start generating log entries.
func (e *Input) Start(persister operator.Persister) error {
	e.Handler = e.handleEvent
	e.Persist = &azure.Persister{DB: persister}
	return e.StartConsumers(context.Background())
}

// Stop will stop generating logs.
func (e *Input) Stop() error {
	return e.StopConsumers()
}
