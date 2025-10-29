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
	"fmt"
	"sync"

	"github.com/observiq/bindplane-otel-collector/receiver/pcapreceiver/internal/capture"
	"github.com/observiq/bindplane-otel-collector/receiver/pcapreceiver/internal/parser"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.uber.org/zap"
)

// pcapReceiver captures network packets and converts them to logs
type pcapReceiver struct {
	config   *Config
	logger   *zap.Logger
	consumer consumer.Logs
	capturer capture.Capturer
	parser   *parser.Parser
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.Mutex
	started  bool
}

func newReceiver(
	config *Config,
	logger *zap.Logger,
	consumer consumer.Logs,
) *pcapReceiver {
	return &pcapReceiver{
		config:   config,
		logger:   logger,
		consumer: consumer,
	}
}

func (r *pcapReceiver) Start(ctx context.Context, _ component.Host) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started {
		return fmt.Errorf("receiver already started")
	}

	// Create capture config
	captureConfig := capture.Config{
		Interface:   r.config.Interface,
		FilePath:    r.config.FilePath,
		SnapLength:  r.config.SnapLength,
		BPFFilter:   r.config.BPFFilter,
		Promiscuous: r.config.Promiscuous,
	}

	// Create capturer
	var err error
	r.capturer, err = capture.NewCapturer(captureConfig)
	if err != nil {
		return fmt.Errorf("failed to create capturer: %w", err)
	}

	// Determine interface name for parser
	interfaceName := r.config.Interface
	if interfaceName == "" {
		if r.config.FilePath != "" {
			interfaceName = r.config.FilePath
		} else {
			interfaceName = "auto"
		}
	}

	// Create parser
	r.parser = parser.NewParser(interfaceName)

	// Create context for packet processing
	var processCtx context.Context
	processCtx, r.cancel = context.WithCancel(ctx)

	// Start capturer
	err = r.capturer.Start(processCtx)
	if err != nil {
		_ = r.capturer.Close()
		return fmt.Errorf("failed to start capturer: %w", err)
	}

	// Start packet processing goroutine
	r.wg.Add(1)
	go r.processPackets(processCtx)

	r.started = true
	r.logger.Info("PCAP receiver started", zap.String("interface", interfaceName))

	return nil
}

func (r *pcapReceiver) Shutdown(_ context.Context) error {
	r.mu.Lock()
	if r.cancel != nil {
		r.cancel()
	}
	r.mu.Unlock()

	// Wait for processing goroutine to finish
	r.wg.Wait()

	// Close capturer
	if r.capturer != nil {
		if err := r.capturer.Close(); err != nil {
			r.logger.Warn("Error closing capturer", zap.Error(err))
		}
	}

	r.logger.Info("PCAP receiver stopped")
	return nil
}

// processPackets reads packets from the capturer and sends them to the consumer
func (r *pcapReceiver) processPackets(ctx context.Context) {
	defer r.wg.Done()

	packetChan := r.capturer.Packets()
	r.logger.Debug("Started processing packets")

	for {
		select {
		case <-ctx.Done():
			r.logger.Debug("Context cancelled, stopping packet processing")
			return

		case packet, ok := <-packetChan:
			if !ok {
				// Channel closed, likely EOF for file-based capture
				r.logger.Debug("Packet channel closed")
				return
			}

			// Parse packet to logs
			logs := r.parser.Parse(packet)

			// Send to consumer
			err := r.consumer.ConsumeLogs(ctx, logs)
			if err != nil {
				r.logger.Error("Failed to consume logs", zap.Error(err))
				// Continue processing other packets
			}
		}
	}
}
