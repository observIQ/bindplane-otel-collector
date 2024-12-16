// Copyright observIQ, Inc.
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

package chronicleexporter

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
	"github.com/shirou/gopsutil/v3/process"
	"go.opentelemetry.io/collector/component"
	semconv "go.opentelemetry.io/collector/semconv/v1.5.0"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type hostMetricsReporter struct {
	set    component.TelemetrySettings
	cancel context.CancelFunc
	wg     sync.WaitGroup
	send   sendMetricsFunc

	mutex       sync.Mutex
	agentID     []byte
	customerID  []byte
	exporterID  string
	namespace   string
	startTime   *timestamppb.Timestamp
	stats       *api.AgentStatsEvent
	logsDropped int64
	logsSent    int64
}

type sendMetricsFunc func(context.Context, *api.BatchCreateEventsRequest) error

func newHostMetricsReporter(cfg *Config, set component.TelemetrySettings, exporterID string, send sendMetricsFunc) (*hostMetricsReporter, error) {
	customerID, err := uuid.Parse(cfg.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("parse customer ID: %w", err)
	}

	agentID := uuid.New()
	if sid, ok := set.Resource.Attributes().Get(semconv.AttributeServiceInstanceID); ok {
		var err error
		agentID, err = uuid.Parse(sid.AsString())
		if err != nil {
			return nil, fmt.Errorf("parse collector ID: %w", err)
		}
	}

	now := timestamppb.Now()
	return &hostMetricsReporter{
		set:        set,
		send:       send,
		agentID:    agentID[:],
		exporterID: exporterID,
		startTime:  now,
		customerID: customerID[:],
		namespace:  cfg.Namespace,
		stats: &api.AgentStatsEvent{
			AgentId:         agentID[:],
			WindowStartTime: now,
			StartTime:       now,
		},
	}, nil
}

func (hmr *hostMetricsReporter) start() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	hmr.cancel = cancel
	hmr.wg.Add(1)
	go func() {
		defer hmr.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				err := hmr.collectHostMetrics()
				if err != nil {
					hmr.set.Logger.Error("Failed to collect host metrics", zap.Error(err))
				}
				request := hmr.getAndReset()
				if err = hmr.send(ctx, request); err != nil {
					hmr.set.Logger.Error("Failed to upload host metrics", zap.Error(err))
				}
			}
		}
	}()
}

func (hmr *hostMetricsReporter) getAndReset() *api.BatchCreateEventsRequest {
	hmr.mutex.Lock()
	defer hmr.mutex.Unlock()

	now := timestamppb.Now()
	batchID := uuid.New()
	source := &api.EventSource{
		CollectorId: chronicleCollectorID[:],
		Namespace:   hmr.namespace,
		CustomerId:  hmr.customerID,
	}

	request := &api.BatchCreateEventsRequest{
		Batch: &api.EventBatch{
			Id:        batchID[:],
			Source:    source,
			Type:      api.EventBatch_AGENT_STATS,
			StartTime: hmr.startTime,
			Events: []*api.Event{
				{
					Timestamp:      now,
					CollectionTime: now,
					Source:         source,
					Payload: &api.Event_AgentStats{
						AgentStats: hmr.stats,
					},
				},
			},
		},
	}

	hmr.resetStats()
	return request
}

func (hmr *hostMetricsReporter) shutdown() {
	if hmr.cancel != nil {
		hmr.cancel()
		hmr.wg.Wait()
	}
}

func (hmr *hostMetricsReporter) resetStats() {
	hmr.stats = &api.AgentStatsEvent{
		ExporterStats: []*api.ExporterStats{
			{
				Name:          hmr.exporterID,
				AcceptedSpans: hmr.logsSent,
				RefusedSpans:  hmr.logsDropped,
			},
		},
		AgentId:         hmr.agentID,
		StartTime:       hmr.startTime,
		WindowStartTime: timestamppb.Now(),
	}
	hmr.logsDropped = 0
	hmr.logsSent = 0
}

func (hmr *hostMetricsReporter) collectHostMetrics() error {
	hmr.mutex.Lock()
	defer hmr.mutex.Unlock()

	// Get the current process using the current PID
	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return fmt.Errorf("get process: %w", err)
	}

	// Collect CPU time used by the process
	cpuTimes, err := proc.Times()
	if err != nil {
		return fmt.Errorf("get cpu times: %w", err)
	}
	totalCPUTime := cpuTimes.User + cpuTimes.System

	// convert to milliseconds
	hmr.stats.ProcessCpuSeconds = int64(totalCPUTime * 1000)

	// Collect memory usage (RSS)
	memInfo, err := proc.MemoryInfo()
	if err != nil {
		return fmt.Errorf("get memory info: %w", err)
	}
	hmr.stats.ProcessMemoryRss = int64(memInfo.RSS / 1024) // Convert bytes to kilobytes

	// Calculate process uptime
	startTimeMs, err := proc.CreateTime()
	if err != nil {
		return fmt.Errorf("get process start time: %w", err)
	}
	startTimeSec := startTimeMs / 1000
	currentTimeSec := time.Now().Unix()
	hmr.stats.ProcessUptime = currentTimeSec - startTimeSec

	return nil
}

func (hmr *hostMetricsReporter) recordSent(count int64) {
	hmr.mutex.Lock()
	defer hmr.mutex.Unlock()
	hmr.logsSent += count
	hmr.stats.LastSuccessfulUploadTime = timestamppb.Now()
}
