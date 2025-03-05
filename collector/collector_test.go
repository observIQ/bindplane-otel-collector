// Copyright Bindplane, Inc.
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

package collector

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

var slowShutdownType = component.MustNewType("slowshutdown")

func TestCollectorRunValid(t *testing.T) {
	ctx := context.Background()

	collector, err := New([]string{"./test/valid.yaml"}, "0.0.0", nil)
	require.NoError(t, err)

	err = collector.Run(ctx)
	require.NoError(t, err)

	status := <-collector.Status()
	require.True(t, status.Running)
	require.NoError(t, status.Err)

	collector.Stop(ctx)
	status = <-collector.Status()
	require.False(t, status.Running)
}

func TestCollectorRunMultiple(t *testing.T) {
	collector, err := New([]string{"./test/valid.yaml"}, "0.0.0", nil)
	require.NoError(t, err)

	for i := 1; i < 5; i++ {
		ctx := context.Background()

		attempt := fmt.Sprintf("Attempt %d", i)
		t.Run(attempt, func(t *testing.T) {
			err := collector.Run(ctx)
			require.NoError(t, err)

			status := <-collector.Status()
			require.True(t, status.Running)
			require.NoError(t, status.Err)

			collector.Stop(ctx)
			status = <-collector.Status()
			require.False(t, status.Running)
		})
	}
}

func TestCollectorRunInvalidConfig(t *testing.T) {
	ctx := context.Background()

	collector, err := New([]string{"./test/invalid.yaml"}, "0.0.0", nil)
	require.NoError(t, err)

	err = collector.Run(ctx)
	require.Error(t, err)

	status := <-collector.Status()
	require.False(t, status.Running)
	require.Error(t, status.Err)
	require.ErrorContains(t, status.Err, "cannot unmarshal the configuration")
}

// There currently exists a limitation in the collector lifecycle regarding context.
// Context is not respected when starting the collector and a collector could run indefinitely
// in this scenario. Once this is addressed, we can readd this test.
//
// func TestCollectorRunCancelledContext(t *testing.T) {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	cancel()

// 	collector := New("./test/valid.yaml", "0.0.0", nil)
// 	err := collector.Run(ctx)
// 	require.EqualError(t, context.Canceled, err.Error())
// }

func TestCollectorRunTwice(t *testing.T) {
	ctx := context.Background()

	collector, err := New([]string{"./test/valid.yaml"}, "0.0.0", nil)
	require.NoError(t, err)

	err = collector.Run(ctx)
	require.NoError(t, err)
	defer collector.Stop(ctx)

	status := <-collector.Status()
	require.True(t, status.Running)
	require.NoError(t, status.Err)

	err = collector.Run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "service already running")

	collector.Stop(ctx)
	status = <-collector.Status()
	require.False(t, status.Running)
}

func TestCollectorRestart(t *testing.T) {
	ctx := context.Background()

	collector, err := New([]string{"./test/valid.yaml"}, "0.0.0", nil)
	require.NoError(t, err)

	err = collector.Run(ctx)
	require.NoError(t, err)

	status := <-collector.Status()
	require.True(t, status.Running)
	require.NoError(t, status.Err)

	err = collector.Restart(ctx)
	require.NoError(t, err)

	status = <-collector.Status()
	require.True(t, status.Running)

	collector.Stop(ctx)
	status = <-collector.Status()
	require.False(t, status.Running)
}

func TestCollectorPrematureStop(t *testing.T) {
	collector, err := New([]string{"./test/valid.yaml"}, "0.0.0", nil)
	require.NoError(t, err)

	collector.Stop(context.Background())
	require.Equal(t, 0, len(collector.Status()))
}

func TestCollectorStopContextTimeout(t *testing.T) {
	col, err := New([]string{"./test/slow_receiver.yaml"}, "0.0.0", nil)
	require.NoError(t, err)

	concreteCol := col.(*collector)
	concreteCol.factories.Receivers[slowShutdownType] = slowShutdownReceiverFactory()

	err = col.Run(context.Background())
	require.NoError(t, err)

	status := <-col.Status()
	require.True(t, status.Running)
	require.NoError(t, status.Err)

	stopCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	t.Cleanup(cancel)

	stopped := make(chan struct{})
	go func() {
		col.Stop(stopCtx)
		close(stopped)
	}()

	select {
	case <-time.After(5 * time.Second):
		t.Fatalf("Shutdown took too long")
	case <-stopped:
	}

	status = <-col.Status()
	require.False(t, status.Running)
}

func TestCollectorRestartContextTimeout(t *testing.T) {
	col, err := New([]string{"./test/slow_receiver.yaml"}, "0.0.0", nil)
	require.NoError(t, err)

	// Replace the restart timeout to be shorter so the test doesn't take a long time.
	oldTimeout := collectorRestartTimeout
	collectorRestartTimeout = 500 * time.Millisecond
	t.Cleanup(func() {
		collectorRestartTimeout = oldTimeout
	})

	concreteCol := col.(*collector)
	concreteCol.factories.Receivers[slowShutdownType] = slowShutdownReceiverFactory()

	err = col.Run(context.Background())
	require.NoError(t, err)
	defer col.Stop(context.Background())

	status := <-col.Status()
	require.True(t, status.Running)
	require.NoError(t, status.Err)

	restarted := make(chan struct{})
	go func() {
		err = col.Restart(context.Background())
		require.NoError(t, err)
		close(restarted)
	}()

	select {
	case <-time.After(5 * time.Second):
		t.Fatalf("Shutdown took too long")
	case <-restarted:
	}

	status = <-col.Status()
	require.True(t, status.Running)

	stopCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	t.Cleanup(cancel)

	stopped := make(chan struct{})
	go func() {
		col.Stop(stopCtx)
		close(stopped)
	}()

	status = <-col.Status()
	require.False(t, status.Running)
}

// slowShutdownReceiver only shutsdown if the shutdown context is cancelled.
func slowShutdownReceiverFactory() receiver.Factory {
	return receiver.NewFactory(slowShutdownType,
		func() component.Config { return &struct{}{} },
		receiver.WithLogs(createLogsSlowShutdownReceiverReceiver, component.StabilityLevelDevelopment),
	)
}

func createLogsSlowShutdownReceiverReceiver(_ context.Context, _ receiver.Settings, _ component.Config, _ consumer.Logs) (receiver.Logs, error) {
	return &slowShutdownReceiver{}, nil
}

// slowShutdownReceiver is a receiver that does not shut down unless it's context is cancelled.
type slowShutdownReceiver struct{}

func (slowShutdownReceiver) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (slowShutdownReceiver) Shutdown(ctx context.Context) error {
	<-ctx.Done()
	return nil
}
