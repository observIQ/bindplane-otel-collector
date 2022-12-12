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

package routereceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestReceiverMetrics(t *testing.T) {
	testCases := []struct {
		name         string
		route        string
		receiverName string
		consumer     consumer.Metrics
		expectedErr  error
	}{
		{
			name:         "metric consumer not defined",
			route:        "test",
			receiverName: "test",
			expectedErr:  errMetricPipelineNotDefined,
		},
		{
			name:         "route not defined",
			route:        "test-1",
			receiverName: "test-2",
			expectedErr:  errRouteNotDefined,
		},
		{
			name:         "valid",
			route:        "test",
			receiverName: "test",
			consumer:     &nopConsumer{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig().(*Config)
			cfg.SetIDName(tc.receiverName)
			set := receivertest.NewNopCreateSettings()
			receiver, err := factory.CreateMetricsReceiver(context.Background(), set, cfg, tc.consumer)
			require.NoError(t, err)

			err = receiver.Start(context.Background(), nil)
			require.NoError(t, err)

			defer receiver.Shutdown(context.Background())

			err = RouteMetrics(context.Background(), tc.route, pmetric.NewMetrics())
			if err != tc.expectedErr {
				t.Errorf("expected error %v, got %v", tc.expectedErr, err)
			}
		})
	}
}

func TestReceiverLogs(t *testing.T) {
	testCases := []struct {
		name         string
		route        string
		receiverName string
		consumer     consumer.Logs
		expectedErr  error
	}{
		{
			name:         "log consumer not defined",
			route:        "test",
			receiverName: "test",
			expectedErr:  errLogPipelineNotDefined,
		},
		{
			name:         "route not defined",
			route:        "test-1",
			receiverName: "test-2",
			expectedErr:  errRouteNotDefined,
		},
		{
			name:         "valid",
			route:        "test",
			receiverName: "test",
			consumer:     &nopConsumer{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig().(*Config)
			cfg.SetIDName(tc.receiverName)
			set := receivertest.NewNopCreateSettings()
			receiver, err := factory.CreateLogsReceiver(context.Background(), set, cfg, tc.consumer)
			require.NoError(t, err)

			err = receiver.Start(context.Background(), nil)
			require.NoError(t, err)

			defer receiver.Shutdown(context.Background())

			err = RouteLogs(context.Background(), tc.route, plog.NewLogs())
			if err != tc.expectedErr {
				t.Errorf("expected error %v, got %v", tc.expectedErr, err)
			}
		})
	}
}

func TestReceiverTraces(t *testing.T) {
	testCases := []struct {
		name         string
		route        string
		receiverName string
		consumer     consumer.Traces
		expectedErr  error
	}{
		{
			name:         "trace consumer not defined",
			route:        "test",
			receiverName: "test",
			expectedErr:  errTracePipelineNotDefined,
		},
		{
			name:         "route not defined",
			route:        "test-1",
			receiverName: "test-2",
			expectedErr:  errRouteNotDefined,
		},
		{
			name:         "valid",
			route:        "test",
			receiverName: "test",
			consumer:     &nopConsumer{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig().(*Config)
			cfg.SetIDName(tc.receiverName)
			set := receivertest.NewNopCreateSettings()
			receiver, err := factory.CreateTracesReceiver(context.Background(), set, cfg, tc.consumer)
			require.NoError(t, err)

			err = receiver.Start(context.Background(), nil)
			require.NoError(t, err)

			defer receiver.Shutdown(context.Background())

			err = RouteTraces(context.Background(), tc.route, ptrace.NewTraces())
			if err != tc.expectedErr {
				t.Errorf("expected error %v, got %v", tc.expectedErr, err)
			}
		})
	}
}

// nopConsumer is a nop consumer.
type nopConsumer struct{}

// ConsumeMetrics implements consumer.Metrics.
func (n *nopConsumer) ConsumeMetrics(_ context.Context, _ pmetric.Metrics) error {
	return nil
}

// ConsumeLogs implements consumer.Logs.
func (n *nopConsumer) ConsumeLogs(_ context.Context, _ plog.Logs) error {
	return nil
}

// ConsumeTraces implements consumer.Traces.
func (n *nopConsumer) ConsumeTraces(_ context.Context, _ ptrace.Traces) error {
	return nil
}

// Capabilities implements consumer.Capabilities.
func (n *nopConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
