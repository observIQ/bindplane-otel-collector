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
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentstatus"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

const contentTypeAttribute = "http.request.header.content-type"

type httpLogsReceiver struct {
	config            *Config
	path              string
	serverSettings    *confighttp.ServerConfig
	telemetrySettings component.TelemetrySettings
	server            *http.Server
	consumer          consumer.Logs
	wg                sync.WaitGroup
	logger            *zap.Logger
}

// newHTTPLogsReceiver returns a newly configured httpLogsReceiver
func newHTTPLogsReceiver(params receiver.Settings, cfg *Config, consumer consumer.Logs) (*httpLogsReceiver, error) {
	return &httpLogsReceiver{
		config:            cfg,
		path:              cfg.Path,
		serverSettings:    &cfg.ServerConfig,
		telemetrySettings: params.TelemetrySettings,
		consumer:          consumer,
		logger:            params.Logger,
	}, nil
}

// Start calls startListening
func (r *httpLogsReceiver) Start(ctx context.Context, host component.Host) error {
	return r.startListening(ctx, host)
}

// startListening starts serve on the server using TLS depending on receiver configuration
func (r *httpLogsReceiver) startListening(ctx context.Context, host component.Host) error {
	r.logger.Debug("starting receiver HTTP server")
	var err error
	r.server, err = r.serverSettings.ToServer(ctx, host, r.telemetrySettings, r)
	if err != nil {
		return fmt.Errorf("to server: %w", err)
	}

	listener, err := r.serverSettings.ToListener(ctx)
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
			componentstatus.ReportStatus(host, componentstatus.NewFatalErrorEvent(err))
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
	if r.server == nil {
		// Nothing to shut down
		return nil
	}

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

	now := pcommon.NewTimestampFromTime(time.Now())

	// parse []byte into plog.Logs
	contentType := req.Header.Get("Content-Type")
	logs, err := r.parsePayloadForContentType(now, payload, contentType)
	if err != nil {
		rw.WriteHeader(http.StatusUnprocessableEntity)
		r.logger.Error("failed to process log payload", zap.Error(err), zap.String("payload", string(payload)))
		return
	}

	// consume logs after processing
	if err := r.consumer.ConsumeLogs(req.Context(), *logs); err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		r.logger.Error("failed to consume logs", zap.Error(err))
		return
	}

	rw.WriteHeader(http.StatusOK)
}

// processJSONLogs transforms the parsed JSON payload into plog.Logs
func (r *httpLogsReceiver) processJSONLogs(now pcommon.Timestamp, logs []map[string]any, contentType string) *plog.Logs {
	pLogs := plog.NewLogs()
	resourceLogs := pLogs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

	for _, log := range logs {
		logRecord := scopeLogs.LogRecords().AppendEmpty()

		logRecord.SetObservedTimestamp(now)

		if err := logRecord.Body().SetEmptyMap().FromRaw(log); err != nil {
			r.logger.Warn("unable to set log body", zap.Error(err))
		}
		logRecord.Attributes().PutStr(contentTypeAttribute, contentType)
	}

	return &pLogs
}

func (r *httpLogsReceiver) parsePayloadForContentType(now pcommon.Timestamp, payload []byte, contentType string) (*plog.Logs, error) {
	if len(payload) == 0 {
		return nil, fmt.Errorf("empty payload")
	}

	if r.config.Raw {
		return r.parsePayloadAsText(now, payload, contentType)
	}

	// for backwards-compatibility, if the content type is not being set, we will treat the payload as JSON if it is valid JSON
	if contentType == "" {
		return r.parsePayloadAsJSON(now, payload, contentType)
	}

	switch {
	case isJSONContentTypeHeader(contentType):
		return r.parsePayloadAsJSON(now, payload, contentType)
	case isTextContentType(contentType):
		return r.parsePayloadAsText(now, payload, contentType)
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}
}

func (r *httpLogsReceiver) parsePayloadAsText(now pcommon.Timestamp, payload []byte, contentType string) (*plog.Logs, error) {
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecord := scopeLogs.LogRecords().AppendEmpty()
	logRecord.Body().SetStr(string(payload))
	logRecord.SetObservedTimestamp(now)
	logRecord.Attributes().PutStr(contentTypeAttribute, contentType)
	return &logs, nil
}

func (r *httpLogsReceiver) parsePayloadAsJSON(now pcommon.Timestamp, payload []byte, contentType string) (*plog.Logs, error) {
	firstChar := seekFirstNonWhitespace(string(payload))
	switch firstChar {
	case "{":
		rawLogObject := map[string]any{}
		if err := json.Unmarshal(payload, &rawLogObject); err != nil {
			return nil, err
		}
		return r.processJSONLogs(now, []map[string]any{rawLogObject}, contentType), nil
	case "[":
		rawLogsArray := []json.RawMessage{}
		if err := json.Unmarshal(payload, &rawLogsArray); err != nil {
			return nil, err
		}
		return r.parseJSONArray(now, rawLogsArray, contentType)
	}
	return nil, fmt.Errorf("malformed JSON payload")
}

// isJSONContentTypeHeader checks if the content type indicates JSON
func isJSONContentTypeHeader(contentType string) bool {
	// Handle content types like "application/json", "application/json; charset=utf-8", etc.
	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return ct == "application/json" || strings.HasSuffix(ct, "+json")
}

func isTextContentType(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0])), "text/")
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
func (r *httpLogsReceiver) parseJSONArray(now pcommon.Timestamp, rawLogs []json.RawMessage, contentType string) (*plog.Logs, error) {
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
	return r.processJSONLogs(now, logs, contentType), nil
}
