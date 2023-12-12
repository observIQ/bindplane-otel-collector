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

package httpreceiver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
	"unicode"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

type httpLogsReceiver struct {
	path              string
	serverSettings    *confighttp.HTTPServerSettings
	telemetrySettings component.TelemetrySettings
	server            *http.Server
	consumer          consumer.Logs
	wg                sync.WaitGroup
	logger            *zap.Logger
}

// newHTTPLogsReceiver returns a newly configured httpLogsReceiver
func newHTTPLogsReceiver(params receiver.CreateSettings, cfg *Config, consumer consumer.Logs) (*httpLogsReceiver, error) {
	return &httpLogsReceiver{
		path:              cfg.Path,
		serverSettings:    &cfg.HTTPServerSettings,
		telemetrySettings: params.TelemetrySettings,
		consumer:          consumer,
		logger:            params.Logger,
	}, nil
}

// Start calls startListening
func (r *httpLogsReceiver) Start(_ context.Context, host component.Host) error {
	return r.startListening(host)
}

// startListening starts serve on the server using TLS depending on receiver configuration
func (r *httpLogsReceiver) startListening(host component.Host) error {
	r.logger.Debug("starting receiver HTTP server")
	var err error
	r.server, err = r.serverSettings.ToServer(host, r.telemetrySettings, r)
	if err != nil {
		return fmt.Errorf("to server: %w", err)
	}

	listener, err := r.serverSettings.ToListener()
	if err != nil {
		return fmt.Errorf("to listener: %w", err)
	}

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.logger.Debug("starting to serve",
			zap.String("address", r.serverSettings.Endpoint),
		)

		err := r.server.Serve(listener)
		r.logger.Debug("Serve done")
		if err != http.ErrServerClosed {
			r.logger.Error("Serve failed", zap.Error(err))
			host.ReportFatalError(err)
		}
	}()

	return nil
}

// Shutdown calls shutdownListener
func (r *httpLogsReceiver) Shutdown(ctx context.Context) error {
	return r.shutdownListener(ctx)
}

// shutdownLIstener tells the server to stop serving and waits for it to stop
func (r *httpLogsReceiver) shutdownListener(ctx context.Context) error {
	r.logger.Debug("shutting down server")

	if err := r.server.Shutdown(ctx); err != nil {
		return err
	}
	r.logger.Debug("waiting for shutdown to complete")
	r.wg.Wait()
	return nil

}

// handleRequest is the function the server uses for requests; calls ConsumeLogs
func (r *httpLogsReceiver) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// path was configured && this req.URL does not match it
	if r.path != "" && req.URL.Path != r.path {
		rw.WriteHeader(http.StatusNotFound)
		r.logger.Debug("received request to path that does not match the configured path", zap.String("request path", req.URL.Path))
		return
	}

	// read in request body
	r.logger.Debug("reading in request body")
	payload, err := io.ReadAll(req.Body)
	if err != nil {
		rw.WriteHeader(http.StatusUnprocessableEntity)
		r.logger.Error("failed to read logs payload", zap.Error(err), zap.String("remote", req.RemoteAddr))
		return
	}

	// parse []byte into map structure
	logs, err := parsePayload(payload)
	if err != nil {
		rw.WriteHeader(http.StatusUnprocessableEntity)
		r.logger.Error("failed to convert log request payload to maps", zap.Error(err), zap.String("payload", string(payload)))
		return
	}

	// consume logs after processing
	if err := r.consumer.ConsumeLogs(req.Context(), r.processLogs(pcommon.NewTimestampFromTime(time.Now()), logs)); err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		r.logger.Error("failed to consume logs", zap.Error(err))
		return
	}

	rw.WriteHeader(http.StatusOK)
}

// processLogs transforms the parsed payload into plog.Logs
func (r *httpLogsReceiver) processLogs(now pcommon.Timestamp, logs []map[string]any) plog.Logs {
	pLogs := plog.NewLogs()
	resourceLogs := pLogs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

	for _, log := range logs {
		logRecord := scopeLogs.LogRecords().AppendEmpty()

		logRecord.SetObservedTimestamp(now)

		if err := logRecord.Body().SetEmptyMap().FromRaw(log); err != nil {
			r.logger.Warn("unable to set log body", zap.Error(err))
		}
	}

	return pLogs
}

// parsePayload transforms the payload into []map[string]any structure
func parsePayload(payload []byte) ([]map[string]any, error) {
	firstChar := seekFirstNonWhitespace(string(payload))
	switch firstChar {
	case "{":
		rawLogObject := map[string]any{}
		if err := json.Unmarshal(payload, &rawLogObject); err != nil {
			return nil, err
		}
		return []map[string]any{rawLogObject}, nil
	case "[":
		rawLogsArray := []json.RawMessage{}
		if err := json.Unmarshal(payload, &rawLogsArray); err != nil {
			return nil, err
		}
		return parseJSONArray(rawLogsArray)
	default:
		return nil, errors.New("unsupported payload format, expected either a JSON object or array")
	}
}

// seekFirstNonWhitespace finds the first non whitespace character of the string
func seekFirstNonWhitespace(s string) string {
	firstChar := ""
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		firstChar = string(r)
		break
	}
	return firstChar
}

// parseJSONArray parses a []json.RawMessage into an array of map[string]any
func parseJSONArray(rawLogs []json.RawMessage) ([]map[string]any, error) {
	logs := make([]map[string]any, 0, len(rawLogs))
	for _, l := range rawLogs {
		if len(l) == 0 {
			continue
		}
		var log map[string]any
		if err := json.Unmarshal(l, &log); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
}
