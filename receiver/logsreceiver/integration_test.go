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

package logsreceiver

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.uber.org/zap"
)

// BenchmarkEmitterToConsumer serves as a benchmark for entries going from the emitter to consumer,
// which follows this path: emitter -> receiver -> converter -> receiver -> consumer
func BenchmarkEmitterToConsumer(b *testing.B) {
	const (
		entryCount = 1_000_000
		hostsCount = 4
	)

	var (
		workerCounts = []int{1, 2, 4, 6, 8}
		entries      = complexEntriesForNDifferentHosts(entryCount, hostsCount)
	)

	for _, wc := range workerCounts {
		b.Run(fmt.Sprintf("worker_count=%d", wc), func(b *testing.B) {
			params := component.ReceiverCreateSettings{
				TelemetrySettings: component.TelemetrySettings{
					Logger: zap.NewNop(),
				},
			}

			factory := NewFactory()
			cfg := factory.CreateDefaultConfig().(*Config)
			cfg.Pipeline = []map[string]interface{}{
				{
					"type": "noop",
				},
			}
			cfg.Converter.WorkerCount = wc

			consumer := &mockLogsConsumer{}
			logsReceiver, err := factory.CreateLogsReceiver(context.Background(), params, cfg, consumer)
			require.NoError(b, err)

			err = logsReceiver.Start(context.Background(), componenttest.NewNopHost())
			require.NoError(b, err)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				consumer.ResetReceivedCount()

				go func() {
					ctx := context.Background()
					for _, e := range entries {
						_ = logsReceiver.(*receiver).emitter.Process(ctx, e)
					}
				}()

				require.Eventually(b,
					func() bool {
						return consumer.Received() == entryCount
					},
					30*time.Second, 5*time.Millisecond, "Did not receive all logs (only received %d)", consumer.Received(),
				)
			}
		})
	}
}

func TestEmitterToConsumer(t *testing.T) {
	const (
		entryCount  = 1_000
		hostsCount  = 4
		workerCount = 2
	)

	entries := complexEntriesForNDifferentHosts(entryCount, hostsCount)

	params := component.ReceiverCreateSettings{
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.NewNop(),
		},
	}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Pipeline = []map[string]interface{}{
		{
			"type": "noop",
		},
	}
	cfg.Converter.WorkerCount = workerCount

	consumer := &mockLogsConsumer{}
	logsReceiver, err := factory.CreateLogsReceiver(context.Background(), params, cfg, consumer)
	require.NoError(t, err)

	err = logsReceiver.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	consumer.ResetReceivedCount()

	go func() {
		ctx := context.Background()
		for _, e := range entries {
			require.NoError(t, logsReceiver.(*receiver).emitter.Process(ctx, e))
		}
	}()

	require.Eventually(t,
		func() bool {
			return consumer.Received() == entryCount
		},
		5*time.Second, 5*time.Millisecond, "Did not receive all logs (only received %d)", consumer.Received(),
	)

	<-time.After(time.Second)

	require.Equal(t, entryCount, consumer.Received())
}
