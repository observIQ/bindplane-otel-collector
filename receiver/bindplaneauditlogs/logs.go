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

package bindplaneauditlogs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

const (
	bindplaneTimeFormat = time.RFC3339
)

// Add a wrapper struct for the API response
type apiResponse struct {
	AuditEvents []AuditLogEvent `json:"auditEvents"`
}

type bindplaneAuditLogsReceiver struct {
	cfg           Config
	client        *http.Client
	consumer      consumer.Logs
	logger        *zap.Logger
	cancel        context.CancelFunc
	wg            *sync.WaitGroup
	lastTimestamp time.Time
	settings      component.TelemetrySettings
}

// newBindplaneAuditLogsReceiver returns a newly configured bindplaneAuditLogsReceiver
func newBindplaneAuditLogsReceiver(cfg *Config, logger *zap.Logger, consumer consumer.Logs) (*bindplaneAuditLogsReceiver, error) {
	return &bindplaneAuditLogsReceiver{
		cfg:           *cfg,
		consumer:      consumer,
		logger:        logger,
		wg:            &sync.WaitGroup{},
		lastTimestamp: time.Now().UTC().Add(-cfg.PollInterval),
	}, nil
}

func (r *bindplaneAuditLogsReceiver) Start(ctx context.Context, host component.Host) error {
	client, err := r.cfg.ToClient(ctx, host, r.settings)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}
	r.client = client

	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.wg.Add(1)
	go r.startPolling(ctx)
	return nil
}

func (r *bindplaneAuditLogsReceiver) startPolling(ctx context.Context) {
	defer r.wg.Done()
	t := time.NewTicker(r.cfg.PollInterval)

	err := r.poll(ctx)
	if err != nil {
		r.logger.Error("there was an error during the first poll", zap.Error(err))
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			err := r.poll(ctx)
			if err != nil {
				r.logger.Error("there was an error during the poll", zap.Error(err))
			}
		}
	}
}

func (r *bindplaneAuditLogsReceiver) poll(ctx context.Context) error {
	logEvents, err := r.getLogs(ctx)
	if err != nil {
		return err
	}
	observedTime := pcommon.NewTimestampFromTime(time.Now())
	logs := r.processLogEvents(observedTime, logEvents)
	if logs.LogRecordCount() > 0 {
		if err := r.consumer.ConsumeLogs(ctx, logs); err != nil {
			return err
		}
	}
	return nil
}

func (r *bindplaneAuditLogsReceiver) getLogs(ctx context.Context) ([]AuditLogEvent, error) {
	var logs []AuditLogEvent
	const timeout = 1 * time.Minute

	reqURL := &url.URL{
		Host: r.cfg.bindplaneURL.Host,
		Path: "/v1/audit-events",
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
	if err != nil {
		err = fmt.Errorf("error creating request: %w", err)
		return nil, err
	}

	query := req.URL.Query()
	query.Add("since", r.lastTimestamp.Format(bindplaneTimeFormat))
	req.URL.RawQuery = query.Encode()

	req.Header.Add("X-Bindplane-Api-Key", r.cfg.APIKey)

	res, err := r.client.Do(req)
	if err != nil {
		err = fmt.Errorf("error making request: %w", err)
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("non-200 response: %s", res.Status)
		return nil, err
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		err = fmt.Errorf("error reading response: %w", err)
		return nil, err
	}

	var response apiResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		err = fmt.Errorf("unable to unmarshal log events: %w", err)
		return nil, err
	}

	logs = response.AuditEvents

	// Sort by timestamp (newest first)
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Timestamp.After(*logs[j].Timestamp)
	})

	// Update lastTimestamp
	if len(logs) > 0 {
		var latestTime time.Time
		for _, event := range logs {
			if event.Timestamp != nil && event.Timestamp.After(latestTime) {
				latestTime = *event.Timestamp
			}
		}
		if !latestTime.IsZero() {
			r.lastTimestamp = latestTime
		}
	}

	return logs, nil
}

func (r *bindplaneAuditLogsReceiver) processLogEvents(observedTime pcommon.Timestamp, logEvents []AuditLogEvent) plog.Logs {
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	resourceLogs.ScopeLogs().AppendEmpty()

	for _, logEvent := range logEvents {
		logRecord := resourceLogs.ScopeLogs().At(0).LogRecords().AppendEmpty()

		// Set timestamps
		logRecord.SetObservedTimestamp(observedTime)
		// Set the timestamp directly since Timestamp is already a *time.Time
		if logEvent.Timestamp != nil {
			logRecord.SetTimestamp(pcommon.NewTimestampFromTime(*logEvent.Timestamp))
		}

		// Set attributes based on the Bindplane audit log format
		attrs := logRecord.Attributes()
		attrs.PutStr("id", logEvent.ID)
		if logEvent.Timestamp != nil {
			attrs.PutStr("timestamp", logEvent.Timestamp.Format(bindplaneTimeFormat))
		}
		attrs.PutStr("resource_name", logEvent.ResourceName)
		attrs.PutStr("description", logEvent.Description)
		attrs.PutStr("resource_kind", string(logEvent.ResourceKind))
		if logEvent.Configuration != "" {
			attrs.PutStr("configuration", logEvent.Configuration)
		}
		attrs.PutStr("action", string(logEvent.Action))
		attrs.PutStr("user", logEvent.User)
		if logEvent.Account != "" {
			attrs.PutStr("account", logEvent.Account)
		}

		resourceAttributes := logRecord.Attributes()
		resourceAttributes.PutStr("bindplane_url", r.cfg.Endpoint)
	}

	return logs
}

func (r *bindplaneAuditLogsReceiver) Shutdown(_ context.Context) error {
	r.logger.Debug("shutting down logs receiver")
	if r.cancel != nil {
		r.cancel()
	}
	r.client.CloseIdleConnections()
	r.wg.Wait()
	return nil
}
