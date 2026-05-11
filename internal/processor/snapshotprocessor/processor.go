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

	"github.com/observiq/bindplane-otel-collector/internal/report"
	"github.com/observiq/bindplane-otel-contrib/pkg/snapshot"
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

	// defaultMaxPayloadBytes is used when a v2 request omits a size cap.
	defaultMaxPayloadBytes = 10 * 1024 * 1024 // 10 MiB
)

// getSnapshotReporter is function for retrieving the SnapshotReporter.
// Meant to be overridden for tests.
var getSnapshotReporter func() *report.SnapshotReporter = report.GetSnapshotReporter

var _ snapshot.Snapshotter = (*report.SnapshotReporter)(nil)

// errShuttingDown is logged by handleSnapshotMessage when shutdown is observed
// while the send was waiting for a pending-send slot.
var errShuttingDown = errors.New("snapshot processor is shutting down")

type snapshotProcessor struct {
	logger      *zap.Logger
	enabled     bool
	snapShotter *report.SnapshotReporter
	processorID component.ID

	// opampExtensionID is the extension that hosts the custom-message
	// registry. Zero value means v2 trigger path is disabled and the
	// processor only serves snapshots through the report manager.
	opampExtensionID component.ID
	handler          opampcustommessages.CustomCapabilityHandler

	started, stopped *atomic.Bool
	doneChan         chan struct{}
	wg               *sync.WaitGroup
}

// newSnapshotProcessor creates a new snapshot processor.
func newSnapshotProcessor(logger *zap.Logger, cfg *Config, processorID component.ID) *snapshotProcessor {
	return &snapshotProcessor{
		logger:           logger,
		enabled:          cfg.Enabled,
		snapShotter:      getSnapshotReporter(),
		processorID:      processorID,
		opampExtensionID: cfg.OpAMP,
		started:          &atomic.Bool{},
		stopped:          &atomic.Bool{},
		doneChan:         make(chan struct{}),
		wg:               &sync.WaitGroup{},
	}
}

// start hooks the processor's v2 trigger path into the OpAMP custom-
// message registry, if one was configured. The v1 trigger path (the
// global report manager) requires no per-instance setup — it is
// already wired through getSnapshotReporter().
func (sp *snapshotProcessor) start(_ context.Context, host component.Host) error {
	if sp.started.Swap(true) {
		return nil
	}
	if sp.opampExtensionID == (component.ID{}) {
		// v1-only mode.
		return nil
	}

	ext, ok := host.GetExtensions()[sp.opampExtensionID]
	if !ok {
		return fmt.Errorf("opamp extension %q does not exist", sp.opampExtensionID)
	}
	registry, ok := ext.(opampcustommessages.CustomCapabilityRegistry)
	if !ok {
		return fmt.Errorf("extension %q is not a custom message registry", sp.opampExtensionID)
	}

	h, err := registry.Register(snapshotCapability)
	if err != nil {
		return fmt.Errorf("register custom capability: %w", err)
	}
	sp.handler = h

	sp.wg.Add(1)
	go sp.processOpAMPMessages()
	return nil
}

func (sp *snapshotProcessor) processOpAMPMessages() {
	defer sp.wg.Done()
	for {
		select {
		case msg := <-sp.handler.Message():
			switch msg.Type {
			case snapshotRequestType:
				sp.handleSnapshotRequest(msg)
			default:
				sp.logger.Warn("received message of unknown type", zap.String("messageType", msg.Type))
			}
		case <-sp.doneChan:
			return
		}
	}
}

// handleSnapshotRequest is the v2 adapter: parse the request, pull
// payload bytes from the same buffer the v1 path is filling, wrap in
// the JSON snapshotReport framing, gzip, and send back.
func (sp *snapshotProcessor) handleSnapshotRequest(cm *protobufs.CustomMessage) {
	var req snapshotRequest
	if err := yaml.Unmarshal(cm.Data, &req); err != nil {
		sp.logger.Error("invalid snapshot request", zap.Error(err))
		return
	}
	if req.Processor != sp.processorID {
		// Message addressed to a different processor instance.
		return
	}
	if req.MaximumPayloadSizeBytes <= 0 {
		req.MaximumPayloadSizeBytes = defaultMaxPayloadBytes
	}

	componentID := sp.processorID.String()

	var report snapshotReport
	switch req.PipelineType {
	case "logs":
		buf := sp.snapShotter.LogBufferFor(componentID)
		if buf == nil {
			report = logsReport(req.SessionID, []byte("[]"))
			break
		}
		payload, err := buf.ConstructPayload(&plog.JSONMarshaler{}, req.SearchQuery, req.MinimumTimestamp, req.MaximumPayloadSizeBytes)
		if err != nil {
			sp.logger.Error("construct logs payload", zap.Error(err))
			return
		}
		report = logsReport(req.SessionID, payload)

	case "metrics":
		buf := sp.snapShotter.MetricBufferFor(componentID)
		if buf == nil {
			report = metricsReport(req.SessionID, []byte("[]"))
			break
		}
		payload, err := buf.ConstructPayload(&pmetric.JSONMarshaler{}, req.SearchQuery, req.MinimumTimestamp, req.MaximumPayloadSizeBytes)
		if err != nil {
			sp.logger.Error("construct metrics payload", zap.Error(err))
			return
		}
		report = metricsReport(req.SessionID, payload)

	case "traces":
		buf := sp.snapShotter.TraceBufferFor(componentID)
		if buf == nil {
			report = tracesReport(req.SessionID, []byte("[]"))
			break
		}
		payload, err := buf.ConstructPayload(&ptrace.JSONMarshaler{}, req.SearchQuery, req.MinimumTimestamp, req.MaximumPayloadSizeBytes)
		if err != nil {
			sp.logger.Error("construct traces payload", zap.Error(err))
			return
		}
		report = tracesReport(req.SessionID, payload)

	default:
		sp.logger.Error("invalid pipeline type in snapshot request", zap.String("pipeline_type", req.PipelineType))
		return
	}

	body, err := json.Marshal(report)
	if err != nil {
		sp.logger.Error("marshal snapshot report", zap.Error(err))
		return
	}
	compressed, err := compress(body)
	if err != nil {
		sp.logger.Error("compress snapshot report", zap.Error(err))
		return
	}

	for {
		ch, err := sp.handler.SendMessage(snapshotReportType, compressed)
		switch {
		case err == nil:
			return
		case errors.Is(err, types.ErrCustomMessagePending):
			select {
			case <-ch:
			case <-sp.doneChan:
				sp.logger.Error("send snapshot report", zap.Error(errShuttingDown))
			}
		default:
			sp.logger.Error("send snapshot report", zap.Error(err))
			return
		}
	}
}

func (sp *snapshotProcessor) processTraces(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	if sp.enabled {
		newTraces := ptrace.NewTraces()
		td.CopyTo(newTraces)
		sp.snapShotter.SaveTraces(sp.processorID.String(), newTraces)
	}
	return td, nil
}

func (sp *snapshotProcessor) processLogs(_ context.Context, ld plog.Logs) (plog.Logs, error) {
	if sp.enabled {
		newLogs := plog.NewLogs()
		ld.CopyTo(newLogs)
		sp.snapShotter.SaveLogs(sp.processorID.String(), newLogs)
	}
	return ld, nil
}

func (sp *snapshotProcessor) processMetrics(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	if sp.enabled {
		newMetrics := pmetric.NewMetrics()
		md.CopyTo(newMetrics)
		sp.snapShotter.SaveMetrics(sp.processorID.String(), newMetrics)
	}
	return md, nil
}

func (sp *snapshotProcessor) stop(ctx context.Context) error {
	if sp.stopped.Swap(true) {
		return nil
	}
	unregisterProcessor(sp.processorID)

	close(sp.doneChan)

	done := make(chan struct{})
	go func() {
		sp.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}

	// Unregister from handler after goroutines have returned
	if sp.handler != nil {
		sp.handler.Unregister()
	}

	return nil
}

// compress gzip compresses the given data.
func compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
