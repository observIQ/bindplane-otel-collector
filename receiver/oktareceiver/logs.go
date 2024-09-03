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

package oktareceiver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/okta/okta-sdk-golang/v2/okta"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// oktaMaxLimit maximum number of log objects returned in one call to Okta API
var oktaMaxLimit = 1000

type oktaLogsReceiver struct {
	cfg      Config
	client   httpClient
	consumer consumer.Logs
	logger   *zap.Logger
	nextURL  string
	cancel   context.CancelFunc
	wg       *sync.WaitGroup
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
	CloseIdleConnections()
}

// newOktaLogsReceiver returns a newly configured oktaLogsReceiver
func newOktaLogsReceiver(cfg *Config, logger *zap.Logger, consumer consumer.Logs) (*oktaLogsReceiver, error) {
	return &oktaLogsReceiver{
		cfg:      *cfg,
		client:   http.DefaultClient,
		consumer: consumer,
		logger:   logger,
		wg:       &sync.WaitGroup{},
	}, nil
}

func (r *oktaLogsReceiver) Start(_ context.Context, _ component.Host) error {
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	r.wg.Add(1)
	go r.startPolling(ctx)
	return nil
}

func (r *oktaLogsReceiver) startPolling(ctx context.Context) {
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

func (r *oktaLogsReceiver) poll(ctx context.Context) error {
	logEvents := r.getLogs(ctx)
	observedTime := pcommon.NewTimestampFromTime(time.Now())
	logs := r.processLogEvents(observedTime, logEvents)
	if logs.LogRecordCount() > 0 {
		if err := r.consumer.ConsumeLogs(ctx, logs); err != nil {
			return err
		}
	}
	return nil
}

func (r *oktaLogsReceiver) getLogs(ctx context.Context) []okta.LogEvent {
	var logs []okta.LogEvent
	var reqURL string
	pollTime := time.Now().UTC()

	// get logs until there isn't any overflow OR we get logs published after initial pollTime
	for {
		if r.nextURL == "" {
			reqURL = "https://" + r.cfg.Domain + "/api/v1/logs"
		} else {
			reqURL = r.nextURL
		}

		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err != nil {
			r.logger.Error("error creating okta api request", zap.Error(err))
			break
		}

		// add query params to the first polling request
		if r.nextURL == "" {
			query := req.URL.Query()
			query.Add("since", pollTime.Add(-r.cfg.PollInterval).Format(OktaTimeFormat))
			query.Add("limit", strconv.Itoa(oktaMaxLimit))
			req.URL.RawQuery = query.Encode()
		}

		req.Header.Add("Authorization", "SSWS "+string(r.cfg.APIToken))

		res, err := r.client.Do(req)
		if err != nil {
			r.logger.Error("error performing okta api request", zap.Error(err))
			break
		}

		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			r.logger.Error("okta logs endpoint returned non-200 statuscode: " + res.Status)
			break
		}

		r.setNextLink(res)

		body, err := io.ReadAll(res.Body)
		if err != nil {
			r.logger.Error("error reading response body", zap.Error(err))
			break
		}

		var curLogs []okta.LogEvent
		err = json.Unmarshal(body, &curLogs)
		if err != nil {
			r.logger.Error("unable to unmarshal log events", zap.Error(err))
			break
		}

		logs = append(logs, curLogs...)

		if len(curLogs) < oktaMaxLimit || curLogs[len(curLogs)-1].Published.After(pollTime) {
			break
		}
	}

	return logs
}

func (r *oktaLogsReceiver) processLogEvents(observedTime pcommon.Timestamp, logEvents []okta.LogEvent) plog.Logs {
	logs := plog.NewLogs()

	// resource attributes
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	resourceLogs.ScopeLogs().AppendEmpty()
	resourceAttributes := resourceLogs.Resource().Attributes()
	resourceAttributes.PutStr("okta.domain", r.cfg.Domain)

	for _, logEvent := range logEvents {
		logRecord := resourceLogs.ScopeLogs().At(0).LogRecords().AppendEmpty()

		// timestamps
		logRecord.SetObservedTimestamp(observedTime)
		timestamp := time.UnixMilli(logEvent.Published.UnixMilli())
		logRecord.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))

		// body
		logEventBytes, err := json.Marshal(logEvent)
		if err != nil {
			r.logger.Error("unable to marshal logEvent", zap.Error(err))
		} else {
			logRecord.Body().SetStr(string(logEventBytes))
		}

		// attributes
		logRecord.Attributes().PutStr("uuid", logEvent.Uuid)
		logRecord.Attributes().PutStr("eventType", logEvent.EventType)
		logRecord.Attributes().PutStr("displayMessage", logEvent.DisplayMessage)
		if logEvent.Outcome != nil {
			logRecord.Attributes().PutStr("outcome.result", logEvent.Outcome.Result)
		}
		if logEvent.Actor != nil {
			logRecord.Attributes().PutStr("actor.id", logEvent.Actor.Id)
			logRecord.Attributes().PutStr("actor.alternateId", logEvent.Actor.AlternateId)
			logRecord.Attributes().PutStr("actor.displayName", logEvent.Actor.DisplayName)
		}
	}

	return logs
}

func (r *oktaLogsReceiver) Shutdown(_ context.Context) error {
	r.logger.Debug("shutting down logs receiver")
	if r.cancel != nil {
		r.cancel()
	}
	r.client.CloseIdleConnections()
	r.wg.Wait()
	return nil
}

func (r *oktaLogsReceiver) setNextLink(res *http.Response) {
	for _, link := range res.Header["Link"] {
		// Split the link into URL and parameters
		parts := strings.Split(strings.TrimSpace(link), ";")
		if len(parts) < 2 {
			continue
		}

		// Check if the "rel" parameter is "next"
		if strings.TrimSpace(parts[1]) == `rel="next"` {
			// Extract and return the URL
			r.nextURL = strings.Trim(parts[0], "<>")
			return
		}
	}
	r.logger.Error("unable to get next link")
}
