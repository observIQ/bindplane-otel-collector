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
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/observiq/bindplane-agent/exporter/chronicleexporter/protos/api"
	"github.com/shirou/gopsutil/v3/process"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type exporterMetrics struct {
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

func newExporterMetrics(agentID, customerID []byte, exporterID, namespace string) *exporterMetrics {
	now := timestamppb.Now()
	return &exporterMetrics{
		agentID:    agentID,
		exporterID: exporterID,
		startTime:  now,
		customerID: customerID,
		namespace:  namespace,
		stats: &api.AgentStatsEvent{
			WindowStartTime: now,
			AgentId:         agentID,
			StartTime:       now,
		},
	}
}

func (cm *exporterMetrics) getAndReset() *api.BatchCreateEventsRequest {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := timestamppb.Now()
	batchID := uuid.New()
	source := &api.EventSource{
		CollectorId: chronicleCollectorID[:],
		Namespace:   cm.namespace,
		CustomerId:  cm.customerID,
	}

	request := &api.BatchCreateEventsRequest{
		Batch: &api.EventBatch{
			Id:        batchID[:],
			Source:    source,
			Type:      api.EventBatch_AGENT_STATS,
			StartTime: cm.startTime,
			Events: []*api.Event{
				{
					Timestamp:      now,
					CollectionTime: now,
					Source:         source,
					Payload: &api.Event_AgentStats{
						AgentStats: cm.stats,
					},
				},
			},
		},
	}

	cm.resetStats()
	return request
}

func (cm *exporterMetrics) resetStats() {
	cm.stats = &api.AgentStatsEvent{
		ExporterStats: []*api.ExporterStats{
			{
				Name:          cm.exporterID,
				AcceptedSpans: cm.logsSent,
				RefusedSpans:  cm.logsDropped,
			},
		},
		AgentId:         cm.agentID,
		StartTime:       cm.startTime,
		WindowStartTime: timestamppb.Now(),
	}
	cm.logsDropped = 0
	cm.logsSent = 0
}

func (cm *exporterMetrics) collectHostMetrics() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

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
	cm.stats.ProcessCpuSeconds = int64(totalCPUTime * 1000)

	// Collect memory usage (RSS)
	memInfo, err := proc.MemoryInfo()
	if err != nil {
		return fmt.Errorf("get memory info: %w", err)
	}
	cm.stats.ProcessMemoryRss = int64(memInfo.RSS / 1024) // Convert bytes to kilobytes

	// Calculate process uptime
	startTimeMs, err := proc.CreateTime()
	if err != nil {
		return fmt.Errorf("get process start time: %w", err)
	}
	startTimeSec := startTimeMs / 1000
	currentTimeSec := time.Now().Unix()
	cm.stats.ProcessUptime = currentTimeSec - startTimeSec

	return nil
}

func (cm *exporterMetrics) updateLastSuccessfulUpload() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.stats.LastSuccessfulUploadTime = timestamppb.Now()
}

func (cm *exporterMetrics) addSentLogs(count int64) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.logsSent += count
}
