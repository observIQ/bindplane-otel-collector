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

package factories

import (
	"errors"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/bearertokenauthextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/tcplogreceiver"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/debugexporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
)

func TestCombineFactories(t *testing.T) {
	testCases := []struct {
		name          string
		receivers     []receiver.Factory
		processors    []processor.Factory
		exporters     []exporter.Factory
		extensions    []extension.Factory
		connectors    []connector.Factory
		expectedError error
	}{
		{
			name:       "With valid combination",
			receivers:  defaultReceivers,
			processors: defaultProcessors,
			exporters:  defaultExporters,
			extensions: defaultExtensions,
			connectors: defaultConnectors,
		},
		{
			name: "With single error",
			receivers: []receiver.Factory{
				tcplogreceiver.NewFactory(),
				tcplogreceiver.NewFactory(),
			},
			expectedError: errors.New(`duplicate component factory "tcplog"`),
		},
		{
			name: "With multiple errors",
			receivers: []receiver.Factory{
				tcplogreceiver.NewFactory(),
				tcplogreceiver.NewFactory(),
			},
			processors: []processor.Factory{
				attributesprocessor.NewFactory(),
				attributesprocessor.NewFactory(),
			},
			exporters: []exporter.Factory{
				debugexporter.NewFactory(),
				debugexporter.NewFactory(),
			},
			extensions: []extension.Factory{
				bearertokenauthextension.NewFactory(),
				bearertokenauthextension.NewFactory(),
			},
			expectedError: errors.New(`duplicate component factory "tcplog"; duplicate component factory "attributes"; duplicate component factory "debug"; duplicate component factory "bearertokenauth"`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factories, err := combineFactories(tc.receivers, tc.processors, tc.exporters, tc.extensions, tc.connectors)

			if tc.expectedError != nil {
				assert.Error(t, tc.expectedError, err)
				assert.Equal(t, tc.expectedError.Error(), err.Error())
				return
			}

			assert.NoError(t, err)

			for _, receiver := range tc.receivers {
				assert.Equal(t, factories.Receivers[receiver.Type()], receiver)
			}

			for _, processor := range tc.processors {
				assert.Equal(t, factories.Processors[processor.Type()], processor)
			}

			for _, exporter := range tc.exporters {
				assert.Equal(t, factories.Exporters[exporter.Type()], exporter)
			}

			for _, extension := range tc.extensions {
				assert.Equal(t, factories.Extensions[extension.Type()], extension)
			}
		})
	}
}

func TestDefaultFactories(t *testing.T) {
	factories, err := DefaultFactories()
	assert.NoError(t, err)

	for _, receiver := range defaultReceivers {
		assert.Equal(t, factories.Receivers[receiver.Type()], receiver)
	}

	for _, processor := range defaultProcessors {
		assert.Equal(t, factories.Processors[processor.Type()], processor)
	}

	for _, exporter := range defaultExporters {
		assert.Equal(t, factories.Exporters[exporter.Type()], exporter)
	}

	for _, extension := range defaultExtensions {
		assert.Equal(t, factories.Extensions[extension.Type()], extension)
	}
}
