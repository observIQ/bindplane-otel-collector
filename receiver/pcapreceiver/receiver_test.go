// Copyright observIQ, Inc.
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

package pcapreceiver

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.uber.org/zap"
)

func TestReceiverLifecycle(t *testing.T) {
	t.Run("start and shutdown gracefully", func(t *testing.T) {
		// Create config with test PCAP file
		pcapPath := filepath.Join("testdata", "capture.pcap")
		_, err := os.Stat(pcapPath)
		if os.IsNotExist(err) {
			t.Skipf("PCAP file not found at %s", pcapPath)
		}

		cfg := &Config{
			Interface:   "",
			FilePath:    pcapPath,
			SnapLength:  65535,
			BPFFilter:   "",
			Promiscuous: false,
		}

		err = cfg.Validate()
		require.NoError(t, err)

		sink := &consumertest.LogsSink{}
		receiver := newReceiver(cfg, zap.NewNop(), sink)
		require.NotNil(t, receiver)

		// Start receiver
		ctx := context.Background()
		err = receiver.Start(ctx, nil)
		require.NoError(t, err, "receiver should start successfully")

		// Give it time to process packets
		time.Sleep(1 * time.Second)

		// Shutdown receiver
		err = receiver.Shutdown(ctx)
		require.NoError(t, err, "receiver should shutdown gracefully")

		// Verify logs were consumed
		logs := sink.AllLogs()
		require.Greater(t, len(logs), 0, "should have consumed logs")
	})

	t.Run("cannot start twice", func(t *testing.T) {
		pcapPath := filepath.Join("testdata", "capture.pcap")
		_, err := os.Stat(pcapPath)
		if os.IsNotExist(err) {
			t.Skipf("PCAP file not found at %s", pcapPath)
		}

		cfg := &Config{
			FilePath:   pcapPath,
			SnapLength: 65535,
		}

		sink := &consumertest.LogsSink{}
		receiver := newReceiver(cfg, zap.NewNop(), sink)

		ctx := context.Background()
		err = receiver.Start(ctx, nil)
		require.NoError(t, err)

		// Try to start again
		err = receiver.Start(ctx, nil)
		require.Error(t, err, "starting twice should fail")

		_ = receiver.Shutdown(ctx)
	})

	t.Run("shutdown without start", func(t *testing.T) {
		cfg := &Config{
			SnapLength: 65535,
		}

		sink := &consumertest.LogsSink{}
		receiver := newReceiver(cfg, zap.NewNop(), sink)

		// Shutdown without starting should not error
		ctx := context.Background()
		err := receiver.Shutdown(ctx)
		require.NoError(t, err, "shutdown without start should not error")
	})
}

func TestReceiver_ProcessPackets_FromFile(t *testing.T) {
	pcapPath := filepath.Join("testdata", "capture.pcap")
	_, err := os.Stat(pcapPath)
	if os.IsNotExist(err) {
		t.Skipf("PCAP file not found at %s", pcapPath)
	}

	cfg := &Config{
		FilePath:   pcapPath,
		SnapLength: 65535,
		BPFFilter:  "",
	}

	err = cfg.Validate()
	require.NoError(t, err)

	sink := &consumertest.LogsSink{}
	receiver := newReceiver(cfg, zap.NewNop(), sink)

	ctx := context.Background()
	err = receiver.Start(ctx, nil)
	require.NoError(t, err)

	// Wait for all packets to be processed
	time.Sleep(2 * time.Second)

	err = receiver.Shutdown(ctx)
	require.NoError(t, err)

	// Verify packets were processed
	logs := sink.AllLogs()
	require.Greater(t, len(logs), 0, "should have processed packets")

	// Verify log records have expected attributes
	for _, ld := range logs {
		resourceLogs := ld.ResourceLogs()
		require.Greater(t, resourceLogs.Len(), 0)

		for i := 0; i < resourceLogs.Len(); i++ {
			scopeLogs := resourceLogs.At(i).ScopeLogs()
			require.Greater(t, scopeLogs.Len(), 0)

			for j := 0; j < scopeLogs.Len(); j++ {
				logRecords := scopeLogs.At(j).LogRecords()
				require.Greater(t, logRecords.Len(), 0)

				for k := 0; k < logRecords.Len(); k++ {
					lr := logRecords.At(k)
					attrs := lr.Attributes()

					// Verify essential attributes
					_, ok := attrs.Get("interface")
					require.True(t, ok, "should have interface attribute")

					_, ok = attrs.Get("packet.size")
					require.True(t, ok, "should have packet.size attribute")

					_, ok = attrs.Get("packet.timestamp")
					require.True(t, ok, "should have packet.timestamp attribute")

					// Should have network layer
					_, ok = attrs.Get("network.protocol")
					require.True(t, ok, "should have network.protocol attribute")

					// Body should not be empty
					require.NotEmpty(t, lr.Body().Str(), "body should not be empty")
				}
			}
		}
	}
}

func TestReceiver_BPFFilter(t *testing.T) {
	pcapPath := filepath.Join("testdata", "capture.pcap")
	_, err := os.Stat(pcapPath)
	if os.IsNotExist(err) {
		t.Skipf("PCAP file not found at %s", pcapPath)
	}

	cfg := &Config{
		FilePath:   pcapPath,
		SnapLength: 65535,
		BPFFilter:  "tcp", // Only TCP packets
	}

	err = cfg.Validate()
	require.NoError(t, err)

	sink := &consumertest.LogsSink{}
	receiver := newReceiver(cfg, zap.NewNop(), sink)

	ctx := context.Background()
	err = receiver.Start(ctx, nil)
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	err = receiver.Shutdown(ctx)
	require.NoError(t, err)

	// Should have processed some packets
	logs := sink.AllLogs()
	if len(logs) > 0 {
		// If packets were captured, verify they're all TCP
		for _, ld := range logs {
			for i := 0; i < ld.ResourceLogs().Len(); i++ {
				for j := 0; j < ld.ResourceLogs().At(i).ScopeLogs().Len(); j++ {
					for k := 0; k < ld.ResourceLogs().At(i).ScopeLogs().At(j).LogRecords().Len(); k++ {
						lr := ld.ResourceLogs().At(i).ScopeLogs().At(j).LogRecords().At(k)
						proto, ok := lr.Attributes().Get("transport.protocol")
						if ok {
							require.Equal(t, "TCP", proto.Str(), "BPF filter should only pass TCP packets")
						}
					}
				}
			}
		}
	}
}

func TestReceiver_InvalidConfig(t *testing.T) {
	t.Run("invalid BPF filter", func(t *testing.T) {
		cfg := &Config{
			SnapLength: 65535,
			BPFFilter:  "invalid syntax!!!",
		}

		err := cfg.Validate()
		require.Error(t, err, "should fail validation with invalid BPF filter")
	})

	t.Run("invalid snap length", func(t *testing.T) {
		cfg := &Config{
			SnapLength: -1,
			BPFFilter:  "",
		}

		err := cfg.Validate()
		require.Error(t, err, "should fail validation with negative snap length")
	})
}

func TestReceiver_ContextCancellation(t *testing.T) {
	pcapPath := filepath.Join("testdata", "capture.pcap")
	_, err := os.Stat(pcapPath)
	if os.IsNotExist(err) {
		t.Skipf("PCAP file not found at %s", pcapPath)
	}

	cfg := &Config{
		FilePath:   pcapPath,
		SnapLength: 65535,
	}

	sink := &consumertest.LogsSink{}
	receiver := newReceiver(cfg, zap.NewNop(), sink)

	ctx, cancel := context.WithCancel(context.Background())
	err = receiver.Start(ctx, nil)
	require.NoError(t, err)

	// Cancel context after short delay
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Give it time to react to cancellation
	time.Sleep(200 * time.Millisecond)

	// Shutdown should still work
	err = receiver.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestNewReceiver(t *testing.T) {
	cfg := &Config{
		SnapLength: 65535,
	}

	sink := &consumertest.LogsSink{}
	receiver := newReceiver(cfg, zap.NewNop(), sink)

	require.NotNil(t, receiver)
	require.Equal(t, cfg, receiver.config)
	require.Equal(t, sink, receiver.consumer)
	require.NotNil(t, receiver.logger)
}
