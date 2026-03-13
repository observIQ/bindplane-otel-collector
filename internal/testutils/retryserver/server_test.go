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

package retryserver_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/observiq/bindplane-otel-collector/internal/testutils/retryserver"
)

// get is a tiny helper that issues a GET to url and returns the response.
func get(t *testing.T, url string) *http.Response {
	t.Helper()
	resp, err := http.Get(url) //nolint:noctx
	require.NoError(t, err)
	return resp
}

// getPath issues a GET to url+path.
func getPath(t *testing.T, base, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(base + path) //nolint:noctx
	require.NoError(t, err)
	return resp
}

func TestNew_DefaultFallback200(t *testing.T) {
	srv := retryserver.New(t, nil)
	resp := get(t, srv.URL())
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, 1, srv.RequestCount())
}

func TestNew_Sequence(t *testing.T) {
	responses := []retryserver.Response{
		{StatusCode: http.StatusTooManyRequests},
		{StatusCode: http.StatusServiceUnavailable},
		{StatusCode: http.StatusOK},
	}

	srv := retryserver.New(t, responses)

	require.Equal(t, http.StatusTooManyRequests, get(t, srv.URL()).StatusCode)
	require.Equal(t, http.StatusServiceUnavailable, get(t, srv.URL()).StatusCode)
	require.Equal(t, http.StatusOK, get(t, srv.URL()).StatusCode)
	// Beyond the sequence → fallback (200)
	require.Equal(t, http.StatusOK, get(t, srv.URL()).StatusCode)

	require.Equal(t, 4, srv.RequestCount())
}

func TestNew_RealWorldRateLimitScenario(t *testing.T) {
	// Simulates consecutive rate-limit responses followed by a successful retry.
	srv := retryserver.New(t, []retryserver.Response{
		{StatusCode: http.StatusTooManyRequests, RetryAfter: "1"},
		{StatusCode: http.StatusTooManyRequests, RetryAfter: "1"},
		{StatusCode: http.StatusOK},
	})

	r1 := get(t, srv.URL())
	require.Equal(t, http.StatusTooManyRequests, r1.StatusCode)
	require.Equal(t, "1", r1.Header.Get("Retry-After"))

	r2 := get(t, srv.URL())
	require.Equal(t, http.StatusTooManyRequests, r2.StatusCode)
	require.Equal(t, "1", r2.Header.Get("Retry-After"))

	r3 := get(t, srv.URL())
	require.Equal(t, http.StatusOK, r3.StatusCode)

	require.Equal(t, 3, srv.RequestCount())
}

func TestNew_RetryAfterHeader(t *testing.T) {
	srv := retryserver.New(t, []retryserver.Response{
		{StatusCode: http.StatusTooManyRequests, RetryAfter: "30"},
	})

	resp := get(t, srv.URL())
	require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	require.Equal(t, "30", resp.Header.Get("Retry-After"))
}

func TestNew_ResponseBody(t *testing.T) {
	srv := retryserver.New(t, []retryserver.Response{
		{StatusCode: http.StatusOK, Body: `{"status":"ok"}`},
	})

	resp := get(t, srv.URL())
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, `{"status":"ok"}`, string(body))
}

func TestNew_CustomHeaders(t *testing.T) {
	srv := retryserver.New(t, []retryserver.Response{
		{
			StatusCode: http.StatusOK,
			Headers:    map[string]string{"X-Custom": "value123"},
		},
	})

	resp := get(t, srv.URL())
	require.Equal(t, "value123", resp.Header.Get("X-Custom"))
}

func TestWithFallback(t *testing.T) {
	srv := retryserver.New(t,
		[]retryserver.Response{
			{StatusCode: http.StatusServiceUnavailable},
		},
		retryserver.WithFallback(retryserver.Response{StatusCode: http.StatusAccepted}),
	)

	require.Equal(t, http.StatusServiceUnavailable, get(t, srv.URL()).StatusCode)
	// Sequence exhausted → custom fallback
	require.Equal(t, http.StatusAccepted, get(t, srv.URL()).StatusCode)
	require.Equal(t, http.StatusAccepted, get(t, srv.URL()).StatusCode)
}

func TestWithRoute_PathSpecificSequence(t *testing.T) {
	srv := retryserver.New(t, nil,
		retryserver.WithRoute("/api/logs", []retryserver.Response{
			{StatusCode: http.StatusTooManyRequests},
			{StatusCode: http.StatusOK},
		}),
	)

	// /api/logs uses the route sequence
	require.Equal(t, http.StatusTooManyRequests, getPath(t, srv.URL(), "/api/logs").StatusCode)
	require.Equal(t, http.StatusOK, getPath(t, srv.URL(), "/api/logs").StatusCode)
	require.Equal(t, 2, srv.RouteRequestCount("/api/logs"))

	// default route (catch-all) is independent
	require.Equal(t, http.StatusOK, get(t, srv.URL()).StatusCode)
	require.Equal(t, 1, srv.RequestCount())
}

func TestWithRoute_MultiplePaths(t *testing.T) {
	srv := retryserver.New(t, nil,
		retryserver.WithRoute("/ingest/logs", []retryserver.Response{
			{StatusCode: http.StatusBadGateway},
			{StatusCode: http.StatusOK},
		}),
		retryserver.WithRoute("/ingest/metrics", []retryserver.Response{
			{StatusCode: http.StatusGatewayTimeout},
			{StatusCode: http.StatusOK},
		}),
	)

	require.Equal(t, http.StatusBadGateway, getPath(t, srv.URL(), "/ingest/logs").StatusCode)
	require.Equal(t, http.StatusOK, getPath(t, srv.URL(), "/ingest/logs").StatusCode)

	require.Equal(t, http.StatusGatewayTimeout, getPath(t, srv.URL(), "/ingest/metrics").StatusCode)
	require.Equal(t, http.StatusOK, getPath(t, srv.URL(), "/ingest/metrics").StatusCode)

	require.Equal(t, 2, srv.RouteRequestCount("/ingest/logs"))
	require.Equal(t, 2, srv.RouteRequestCount("/ingest/metrics"))
}

func TestWithRouteFallback(t *testing.T) {
	srv := retryserver.New(t, nil,
		retryserver.WithRoute("/api/data",
			[]retryserver.Response{
				{StatusCode: http.StatusServiceUnavailable},
			},
			retryserver.WithRouteFallback(retryserver.Response{StatusCode: http.StatusCreated}),
		),
	)

	require.Equal(t, http.StatusServiceUnavailable, getPath(t, srv.URL(), "/api/data").StatusCode)
	// Route sequence exhausted → route-specific fallback
	require.Equal(t, http.StatusCreated, getPath(t, srv.URL(), "/api/data").StatusCode)
}

func TestReset(t *testing.T) {
	srv := retryserver.New(t, []retryserver.Response{
		{StatusCode: http.StatusTooManyRequests},
		{StatusCode: http.StatusOK},
	})

	require.Equal(t, http.StatusTooManyRequests, get(t, srv.URL()).StatusCode)
	require.Equal(t, http.StatusOK, get(t, srv.URL()).StatusCode)
	require.Equal(t, 2, srv.RequestCount())

	srv.Reset()
	require.Equal(t, 0, srv.RequestCount())

	// Sequence replays from the beginning after reset
	require.Equal(t, http.StatusTooManyRequests, get(t, srv.URL()).StatusCode)
	require.Equal(t, 1, srv.RequestCount())
}

func TestReset_WithRoutes(t *testing.T) {
	srv := retryserver.New(t,
		[]retryserver.Response{{StatusCode: http.StatusBadGateway}},
		retryserver.WithRoute("/api", []retryserver.Response{
			{StatusCode: http.StatusServiceUnavailable},
		}),
	)

	get(t, srv.URL())
	getPath(t, srv.URL(), "/api")
	require.Equal(t, 1, srv.RequestCount())
	require.Equal(t, 1, srv.RouteRequestCount("/api"))

	srv.Reset()
	require.Equal(t, 0, srv.RequestCount())
	require.Equal(t, 0, srv.RouteRequestCount("/api"))
}

func TestRouteRequestCount_UnknownPath(t *testing.T) {
	srv := retryserver.New(t, nil)
	require.Equal(t, 0, srv.RouteRequestCount("/never-registered"))
}

func TestGatewayErrorScenarios(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"502 Bad Gateway", http.StatusBadGateway},
		{"503 Service Unavailable", http.StatusServiceUnavailable},
		{"504 Gateway Timeout", http.StatusGatewayTimeout},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := retryserver.New(t, []retryserver.Response{
				{StatusCode: tc.statusCode},
				{StatusCode: tc.statusCode},
				{StatusCode: http.StatusOK},
			})

			require.Equal(t, tc.statusCode, get(t, srv.URL()).StatusCode)
			require.Equal(t, tc.statusCode, get(t, srv.URL()).StatusCode)
			require.Equal(t, http.StatusOK, get(t, srv.URL()).StatusCode)
			require.Equal(t, 3, srv.RequestCount())
		})
	}
}
