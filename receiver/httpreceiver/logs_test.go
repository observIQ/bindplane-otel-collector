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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/plogtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap/zaptest"
)

// tests parsePayload() and processLogs()
func TestPayloadToLogRecord(t *testing.T) {
	testCases := []struct {
		desc         string
		payload      string
		expectedErr  error
		expectedLogs func(*testing.T, string) plog.Logs
	}{
		{
			desc:         "simple pass",
			payload:      `[{"message": "2023/11/06 08:09:10 Generic event", "status": "info", "timestamp": 1699276151086, "hostname": "Dakotas-MBP-2.hsd1.mi.comcast.net", "service": "custom_file", "ddsource": "my_app", "ddtags": "filename:dd-log-file.log"}]`,
			expectedLogs: expectedLogs,
		},
		{
			desc:         "multiple pass",
			payload:      `[{"message": "2023/11/06 08:09:10 Generic event", "status": "info", "timestamp": 1699276151086, "hostname": "Dakotas-MBP-2.hsd1.mi.comcast.net", "service": "custom_file", "ddsource": "my_app", "ddtags": "filename:dd-log-file.log"}, {"message": "2023/11/06 08:10:10 Generic event", "status": "info", "timestamp": 1699276151086, "hostname": "Dakotas-MBP-2.hsd1.mi.comcast.net", "service": "custom_file", "ddsource": "my_app", "ddtags": "filename:dd-log-file.log"}]`,
			expectedLogs: expectedLogs,
		},
		{
			desc:         "nested json",
			payload:      `[{"status": "info", "timestamp": "8402734234", "message": {"data": "hello world"}, "hostname": "localhost"}]`,
			expectedLogs: expectedLogs,
		},
		{
			desc:    "object json",
			payload: `{"status": "info", "timestamp": "8402734234", "message": {"data": "hello world"}, "hostname": "localhost"}`,
			expectedLogs: func(t *testing.T, s string) plog.Logs {
				logs := plog.NewLogs()
				resourceLogs := logs.ResourceLogs().AppendEmpty()
				scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

				rawLog := json.RawMessage{}
				require.NoError(t, json.Unmarshal([]byte(s), &rawLog))

				logRecord := scopeLogs.LogRecords().AppendEmpty()
				logRecord.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))

				var log map[string]any
				require.NoError(t, json.Unmarshal(rawLog, &log))
				require.NoError(t, logRecord.Body().SetEmptyMap().FromRaw(log))

				return logs
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			r := newReceiver(t, &Config{
				Path: "",
				ServerConfig: confighttp.ServerConfig{
					Endpoint:   "localhost:12345",
					TLSSetting: &configtls.TLSServerSetting{},
				},
			}, &consumertest.LogsSink{})
			var logs plog.Logs
			raw, err := parsePayload([]byte(tc.payload))
			if err == nil {
				logs = r.processLogs(pcommon.NewTimestampFromTime(time.Now()), raw)
			}

			if tc.expectedErr != nil {
				require.Error(t, err)
				require.Nil(t, logs)
				require.EqualError(t, err, tc.expectedErr.Error())
			} else {
				require.NoError(t, err)
				require.NotNil(t, logs)
				require.NoError(t, plogtest.CompareLogs(tc.expectedLogs(t, tc.payload), logs, plogtest.IgnoreObservedTimestamp()))
			}
		})
	}
}

func expectedLogs(t *testing.T, payload string) plog.Logs {
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

	rawLogs := []json.RawMessage{}
	require.NoError(t, json.Unmarshal([]byte(payload), &rawLogs))

	for _, l := range rawLogs {
		logRecord := scopeLogs.LogRecords().AppendEmpty()

		logRecord.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))

		var log map[string]any
		require.NoError(t, json.Unmarshal(l, &log))
		require.NoError(t, logRecord.Body().SetEmptyMap().FromRaw(log))
	}

	return logs
}

// tests handleRequest()
func TestServeHTTP(t *testing.T) {
	testCases := []struct {
		desc               string
		cfg                *Config
		request            *http.Request
		expectedStatusCode int
		logExpected        bool
		consumerFailure    bool
	}{
		{
			desc: "simple",
			cfg: &Config{
				Path: "",
				ServerConfig: confighttp.ServerConfig{
					Endpoint:   "localhost:12345",
					TLSSetting: &configtls.TLSServerSetting{},
				},
			},
			request: &http.Request{
				Method: "POST",
				URL:    &url.URL{},
				Header: map[string][]string{
					textproto.CanonicalMIMEHeaderKey("Content-Encoding"): {"identity"},
					textproto.CanonicalMIMEHeaderKey("Content-Type"):     {"application/json"},
				},
				Body: io.NopCloser(bytes.NewBufferString(`[{"message": "2023/11/06 08:09:10 Generic event", "status": "info", "timestamp": 1699276151086, "hostname": "Dakotas-MBP-2.hsd1.mi.comcast.net", "service": "custom_file", "ddsource": "my_app", "ddtags": "filename:dd-log-file.log"}]`)),
			},
			expectedStatusCode: http.StatusOK,
			logExpected:        true,
			consumerFailure:    false,
		},
		{
			desc: "with path",
			cfg: &Config{
				Path: "/logs",
				ServerConfig: confighttp.ServerConfig{
					Endpoint:   "localhost:12345",
					TLSSetting: &configtls.TLSServerSetting{},
				},
			},
			request: &http.Request{
				Method: "POST",
				URL: &url.URL{
					Path: "/logs",
				},
				Header: map[string][]string{
					textproto.CanonicalMIMEHeaderKey("Content-Encoding"): {"identity"},
					textproto.CanonicalMIMEHeaderKey("Content-Type"):     {"application/json"},
				},
				Body: io.NopCloser(bytes.NewBufferString(`[{"message": "2023/11/06 08:09:10 Generic event", "status": "info", "timestamp": 1699276151086, "hostname": "Dakotas-MBP-2.hsd1.mi.comcast.net", "service": "custom_file", "ddsource": "my_app", "ddtags": "filename:dd-log-file.log"}]`)),
			},
			expectedStatusCode: http.StatusOK,
			logExpected:        true,
			consumerFailure:    false,
		},
		{
			desc: "bad path",
			cfg: &Config{
				Path: "/logs",
				ServerConfig: confighttp.ServerConfig{
					Endpoint:   "localhost:12345",
					TLSSetting: &configtls.TLSServerSetting{},
				},
			},
			request: &http.Request{
				Method: "POST",
				URL: &url.URL{
					Path: "/metrics",
				},
				Header: map[string][]string{
					textproto.CanonicalMIMEHeaderKey("Content-Encoding"): {"identity"},
					textproto.CanonicalMIMEHeaderKey("Content-Type"):     {"application/json"},
				},
				Body: io.NopCloser(bytes.NewBufferString(`[{"message": "2023/11/06 08:09:10 Generic event", "status": "info", "timestamp": 1699276151086, "hostname": "Dakotas-MBP-2.hsd1.mi.comcast.net", "service": "custom_file", "ddsource": "my_app", "ddtags": "filename:dd-log-file.log"}]`)),
			},
			expectedStatusCode: http.StatusNotFound,
			logExpected:        false,
			consumerFailure:    false,
		},
		{
			desc: "bad json, parse fails",
			cfg: &Config{
				Path: "/logs",
				ServerConfig: confighttp.ServerConfig{
					Endpoint:   "localhost:12345",
					TLSSetting: &configtls.TLSServerSetting{},
				},
			},
			request: &http.Request{
				Method: "POST",
				URL: &url.URL{
					Path: "/logs",
				},
				Header: map[string][]string{
					textproto.CanonicalMIMEHeaderKey("Content-Encoding"): {"identity"},
					textproto.CanonicalMIMEHeaderKey("Content-Type"):     {"application/json"},
				},
				Body: io.NopCloser(bytes.NewBufferString(`hello world`)),
			},
			expectedStatusCode: http.StatusUnprocessableEntity,
			logExpected:        false,
			consumerFailure:    false,
		},
		{
			desc: "consumer fails",
			cfg: &Config{
				Path: "/logs",
				ServerConfig: confighttp.ServerConfig{
					Endpoint:   "localhost:12345",
					TLSSetting: &configtls.TLSServerSetting{},
				},
			},
			request: &http.Request{
				Method: "POST",
				URL: &url.URL{
					Path: "/logs",
				},
				Header: map[string][]string{
					textproto.CanonicalMIMEHeaderKey("Content-Encoding"): {"identity"},
					textproto.CanonicalMIMEHeaderKey("Content-Type"):     {"application/json"},
				},
				Body: io.NopCloser(bytes.NewBufferString(`[{"message": "2023/11/06 08:09:10 Generic event", "status": "info", "timestamp": 1699276151086, "hostname": "Dakotas-MBP-2.hsd1.mi.comcast.net", "service": "custom_file", "ddsource": "my_app", "ddtags": "filename:dd-log-file.log"}]`)),
			},
			expectedStatusCode: http.StatusInternalServerError,
			logExpected:        false,
			consumerFailure:    true,
		},
		{
			desc: "connectivity test",
			cfg: &Config{
				Path: "/logs",
				ServerConfig: confighttp.ServerConfig{
					Endpoint:   "localhost:12345",
					TLSSetting: &configtls.TLSServerSetting{},
				},
			},
			request: &http.Request{
				Method: "POST",
				URL: &url.URL{
					Path: "/logs",
				},
				Header: map[string][]string{
					textproto.CanonicalMIMEHeaderKey("Content-Encoding"): {"identity"},
					textproto.CanonicalMIMEHeaderKey("Content-Type"):     {"application/json"},
				},
				Body: io.NopCloser(bytes.NewBufferString(`{}`)),
			},
			expectedStatusCode: http.StatusOK,
			logExpected:        true,
			consumerFailure:    false,
		},
		{
			desc: "simple; json object",
			cfg: &Config{
				Path: "",
				ServerConfig: confighttp.ServerConfig{
					Endpoint:   "localhost:12345",
					TLSSetting: &configtls.TLSServerSetting{},
				},
			},
			request: &http.Request{
				Method: "POST",
				URL:    &url.URL{},
				Header: map[string][]string{
					textproto.CanonicalMIMEHeaderKey("Content-Encoding"): {"identity"},
					textproto.CanonicalMIMEHeaderKey("Content-Type"):     {"application/json"},
				},
				Body: io.NopCloser(bytes.NewBufferString(`{"message": "2023/11/06 08:09:10 Generic event", "status": "info", "timestamp": 1699276151086, "hostname": "Dakotas-MBP-2.hsd1.mi.comcast.net", "service": "custom_file", "ddsource": "my_app", "ddtags": "filename:dd-log-file.log"}`)),
			},
			expectedStatusCode: http.StatusOK,
			logExpected:        true,
			consumerFailure:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			var consumer consumer.Logs
			if tc.consumerFailure {
				consumer = consumertest.NewErr(errors.New("consumer failed"))
			} else {
				consumer = &consumertest.LogsSink{}
			}

			r := newReceiver(t, tc.cfg, consumer)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, tc.request)

			assert.Equal(t, tc.expectedStatusCode, rec.Code, "status codes are not equal")
			if !tc.consumerFailure {
				if tc.logExpected {
					assert.Equal(t, 1, consumer.(*consumertest.LogsSink).LogRecordCount(), "did not receive log record")
				} else {
					assert.Equal(t, 0, consumer.(*consumertest.LogsSink).LogRecordCount(), "Received log record when it should have been dropped")
				}
			}
		})
	}
}

func TestShutdownNoServer(t *testing.T) {
	// test that shutdown without a start does not error or panic
	recv := newReceiver(t, &Config{
		ServerConfig: confighttp.ServerConfig{
			Endpoint: "localhost:12345",
		},
	}, consumertest.NewNop())

	require.NoError(t, recv.Shutdown(context.Background()))
}

func newReceiver(t *testing.T, cfg *Config, c consumer.Logs) *httpLogsReceiver {
	s := receivertest.NewNopCreateSettings()
	s.Logger = zaptest.NewLogger(t)
	r, err := newHTTPLogsReceiver(s, cfg, c)
	require.NoError(t, err)
	return r
}
