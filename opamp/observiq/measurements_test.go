// Copyright Bindplane, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package observiq

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/snappy"
	"github.com/observiq/bindplane-otel-collector/internal/measurements"
	"github.com/observiq/bindplane-otel-collector/opamp/mocks"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
)

func TestMeasurementsSender(t *testing.T) {
	t.Run("Test emits metrics", func(t *testing.T) {
		dataChan := make(chan []byte, 1)
		client := mocks.NewMockOpAMPClient(t)
		client.On("SendCustomMessage", mock.Anything).Run(func(args mock.Arguments) {
			cm := args.Get(0).(*protobufs.CustomMessage)
			select {
			case dataChan <- cm.Data:
			default:
			}

		}).Return(make(chan struct{}), nil)

		mp := metric.NewMeterProvider()
		defer mp.Shutdown(context.Background())

		processorID := "throughputmeasurement/1"

		tm, err := measurements.NewThroughputMeasurements(mp, processorID, map[string]string{})
		require.NoError(t, err)

		m, err := golden.ReadMetrics(filepath.Join("testdata", "metrics", "host-metrics.yaml"))
		require.NoError(t, err)

		tm.AddMetrics(context.Background(), m)

		reg := measurements.NewResettableThroughputMeasurementsRegistry(false)
		require.NoError(t, reg.RegisterThroughputMeasurements(processorID, tm))

		ms := newMeasurementsSender(zap.NewNop(), reg, client, 1*time.Millisecond, nil)
		ms.Start()

		select {
		case <-time.After(1 * time.Second):
			require.FailNow(t, "timed out waiting for metrics payload")
		case d := <-dataChan:
			decoded, err := snappy.Decode(nil, d)
			require.NoError(t, err)

			um := &pmetric.ProtoUnmarshaler{}
			actualMetrics, err := um.UnmarshalMetrics(decoded)
			require.NoError(t, err)

			expectedMetrics, err := golden.ReadMetrics(filepath.Join("testdata", "metrics", "expected-throughput.yaml"))
			require.NoError(t, err)

			require.NoError(t, pmetrictest.CompareMetrics(expectedMetrics, actualMetrics, pmetrictest.IgnoreTimestamp()))
		}

		ms.Stop()
	})

	t.Run("Test set interval", func(t *testing.T) {
		dataChan := make(chan []byte, 1)
		client := mocks.NewMockOpAMPClient(t)
		client.On("SendCustomMessage", mock.Anything).Run(func(args mock.Arguments) {
			cm := args.Get(0).(*protobufs.CustomMessage)
			select {
			case dataChan <- cm.Data:
			default:
			}
		}).Return(make(chan struct{}), nil)

		mp := metric.NewMeterProvider()
		defer mp.Shutdown(context.Background())

		processorID := "throughputmeasurement/1"

		tm, err := measurements.NewThroughputMeasurements(mp, processorID, map[string]string{})
		require.NoError(t, err)

		m, err := golden.ReadMetrics(filepath.Join("testdata", "metrics", "host-metrics.yaml"))
		require.NoError(t, err)

		tm.AddMetrics(context.Background(), m)

		reg := measurements.NewResettableThroughputMeasurementsRegistry(false)
		reg.RegisterThroughputMeasurements(processorID, tm)

		ms := newMeasurementsSender(zap.NewNop(), reg, client, 5*time.Hour, nil)
		ms.Start()

		// Wait 200 ms and ensure no data emitted
		time.Sleep(200 * time.Millisecond)

		require.Len(t, dataChan, 0)

		// Set time to 1ms. We should see data emit quickly after.
		ms.SetInterval(1 * time.Millisecond)

		select {
		case <-time.After(1 * time.Second):
			require.FailNow(t, "timed out waiting for metrics payload")
		case d := <-dataChan:
			decoded, err := snappy.Decode(nil, d)
			require.NoError(t, err)

			um := &pmetric.ProtoUnmarshaler{}
			actualMetrics, err := um.UnmarshalMetrics(decoded)
			require.NoError(t, err)

			expectedMetrics, err := golden.ReadMetrics(filepath.Join("testdata", "metrics", "expected-throughput.yaml"))
			require.NoError(t, err)

			require.NoError(t, pmetrictest.CompareMetrics(expectedMetrics, actualMetrics, pmetrictest.IgnoreTimestamp()))
		}

		ms.Stop()
	})
}
