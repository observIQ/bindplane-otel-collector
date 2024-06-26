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

package snapshotprocessor

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/observiq/bindplane-agent/internal/report/snapshot"
	"github.com/open-telemetry/opamp-go/client/types"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/opampcustommessages"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	snapshotCapability  = "com.bindplane.snapshot"
	snapshotRequestType = "requestSnapshot"
	snapshotReportType  = "reportSnapshot"
)

type snapshotProcessor struct {
	logger *zap.Logger

	processorID      component.ID
	enabled          bool
	opampExtensionID component.ID

	customCapabilityHandler opampcustommessages.CustomCapabilityHandler

	logBuffer    *snapshot.LogBuffer
	metricBuffer *snapshot.MetricBuffer
	traceBuffer  *snapshot.TraceBuffer

	started  *atomic.Bool
	stopped  *atomic.Bool
	doneChan chan struct{}
	wg       *sync.WaitGroup
}

// newSnapshotProcessor creates a new snapshot processor
func newSnapshotProcessor(logger *zap.Logger, cfg *Config, processorID component.ID) *snapshotProcessor {
	return &snapshotProcessor{
		logger: logger,

		enabled:          cfg.Enabled,
		processorID:      processorID,
		opampExtensionID: cfg.OpAMP,

		logBuffer:    snapshot.NewLogBuffer(100),
		metricBuffer: snapshot.NewMetricBuffer(100),
		traceBuffer:  snapshot.NewTraceBuffer(100),

		started:  &atomic.Bool{},
		stopped:  &atomic.Bool{},
		doneChan: make(chan struct{}),
		wg:       &sync.WaitGroup{},
	}
}

func (sp *snapshotProcessor) start(_ context.Context, host component.Host) error {
	if sp.started.Swap(true) {
		// Start logic should only be run once
		return nil
	}

	ext, ok := host.GetExtensions()[sp.opampExtensionID]
	if !ok {
		return fmt.Errorf("opamp extension %q does not exist", sp.opampExtensionID)
	}

	registry, ok := ext.(opampcustommessages.CustomCapabilityRegistry)
	if !ok {
		return fmt.Errorf("extension %q is not an custom message registry", sp.opampExtensionID)
	}

	var err error
	sp.customCapabilityHandler, err = registry.Register(snapshotCapability)
	if err != nil {
		return fmt.Errorf("register custom capability: %w", err)
	}

	sp.wg.Add(1)
	go sp.processOpAMPMessages(sp.customCapabilityHandler)

	return nil
}

func (sp *snapshotProcessor) processOpAMPMessages(o opampcustommessages.CustomCapabilityHandler) {
	defer sp.wg.Done()
	for {
		select {
		case msg := <-o.Message():
			switch msg.Type {
			case snapshotRequestType:
				sp.logger.Debug("got snapshot request message")
				sp.processSnapshotRequest(msg)
			default:
				sp.logger.Warn("Received message of unknown type.", zap.String("messageType", msg.Type))
			}
			continue
		case <-sp.doneChan:
			return
		}
	}
}

func (sp *snapshotProcessor) processSnapshotRequest(cm *protobufs.CustomMessage) {
	var req snapshotRequest
	err := yaml.Unmarshal(cm.Data, &req)
	if err != nil {
		sp.logger.Error("Got invalid snapshot request.", zap.Error(err))
		return
	}

	if req.Processor != sp.processorID {
		// // message is for a difference processor, skip.
		return
	}

	sp.logger.Debug("Processor ID on snapshot message matched", zap.Stringer("processor_id", req.Processor))

	var report snapshotReport
	switch req.PipelineType {
	case "logs":
		telemetryPayload, err := sp.logBuffer.ConstructPayload(&plog.JSONMarshaler{}, req.SearchQuery, req.MinimumTimestamp)
		if err != nil {
			sp.logger.Error("Failed to construct snapshot payload.", zap.Error(err))
			return
		}

		report = logsReport(req.SessionID, telemetryPayload)

	case "metrics":
		telemetryPayload, err := sp.metricBuffer.ConstructPayload(&pmetric.JSONMarshaler{}, req.SearchQuery, req.MinimumTimestamp)
		if err != nil {
			sp.logger.Error("Failed to construct metrics snapshot payload.", zap.Error(err))
			return
		}

		report = metricsReport(req.SessionID, telemetryPayload)

	case "traces":
		telemetryPayload, err := sp.traceBuffer.ConstructPayload(&ptrace.JSONMarshaler{}, req.SearchQuery, req.MinimumTimestamp)
		if err != nil {
			sp.logger.Error("Failed to construct traces payload.", zap.Error(err))
			return
		}

		report = tracesReport(req.SessionID, telemetryPayload)

	default:
		sp.logger.Error("Invalid pipeline type in snapshot request.", zap.String("PipelineType", req.PipelineType))
		return
	}

	sp.logger.Info("responding to report request", zap.String("session", req.SessionID))

	response, err := json.Marshal(report)
	if err != nil {
		sp.logger.Error("Could not marshal snapshot report.", zap.Error(err))
		return
	}

	compressedResponse, err := compress(response)
	if err != nil {
		sp.logger.Error("Failed to compress snapshot payload.", zap.Error(err))
		return
	}

	for {
		msgSendChan, err := sp.customCapabilityHandler.SendMessage(snapshotReportType, compressedResponse)
		switch {
		case err == nil: // Message is scheduled to send
			sp.logger.Debug("Message scheduled")
			return

		case errors.Is(err, types.ErrCustomMessagePending):
			// Wait until message is ready to send, then try again
			sp.logger.Debug("Custom message pending, will try sending again after message is clear.")
			<-msgSendChan

		default:
			sp.logger.Error("Failed to send snapshot payload message.", zap.Error(err))
			return
		}
	}
}

func (sp *snapshotProcessor) processTraces(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	if sp.enabled {
		newTraces := ptrace.NewTraces()
		td.CopyTo(newTraces)
		sp.traceBuffer.Add(newTraces)
	}

	return td, nil
}

func (sp *snapshotProcessor) processLogs(_ context.Context, ld plog.Logs) (plog.Logs, error) {
	if sp.enabled {
		newLogs := plog.NewLogs()
		ld.CopyTo(newLogs)
		sp.logBuffer.Add(newLogs)
	}

	return ld, nil
}

func (sp *snapshotProcessor) processMetrics(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	if sp.enabled {
		newMetrics := pmetric.NewMetrics()
		md.CopyTo(newMetrics)
		sp.metricBuffer.Add(newMetrics)
	}

	return md, nil
}

func (sp *snapshotProcessor) stop(ctx context.Context) error {
	if sp.stopped.Swap(true) {
		// Stop logic should only be run once
		return nil
	}

	unregisterProcessor(sp.processorID)

	if sp.customCapabilityHandler != nil {
		sp.customCapabilityHandler.Unregister()
	}

	if sp.doneChan != nil {
		close(sp.doneChan)
	}

	waitgroupDone := make(chan struct{})
	go func() {
		sp.wg.Wait()
		close(waitgroupDone)
	}()

	select {
	case <-waitgroupDone:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// compress gzip compresses the input data
func compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, err := w.Write(data)
	if err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
