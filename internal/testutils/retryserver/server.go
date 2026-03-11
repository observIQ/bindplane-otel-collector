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

// Package retryserver provides a WireMock-style configurable HTTP mock server
// for testing retry behavior in exporters. It simulates real-world failure
// scenarios — rate limiting, gateway errors, transient failures — by responding
// with a predefined sequence of HTTP responses, after which it falls back to a
// default response.
//
// # Usage
//
// Create a server with a sequence of responses and point your exporter at it:
//
//	srv := retryserver.New(t, []retryserver.Response{
//	    {StatusCode: 429, RetryAfter: "1"},
//	    {StatusCode: 429},
//	    {StatusCode: 200},
//	})
//	// srv.URL() returns the base URL to configure your exporter against
//	// After the test, check srv.RequestCount() to assert retry attempts
//
// # Routing
//
// By default the server matches all paths. Use WithRoute to register
// path-specific sequences (useful for exporters that use distinct endpoints):
//
//	srv := retryserver.New(t, nil,
//	    retryserver.WithRoute("/api/logs", []retryserver.Response{
//	        {StatusCode: 503},
//	        {StatusCode: 200},
//	    }),
//	)
package retryserver //import "github.com/observiq/bindplane-otel-collector/internal/testutils/retryserver"

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// Response defines a single HTTP response in a sequence.
type Response struct {
	// StatusCode is the HTTP status code to return. Defaults to 200 if zero.
	StatusCode int
	// RetryAfter is the optional value for the Retry-After response header.
	// Accepts seconds as a string ("2") or an HTTP date (RFC1123 format).
	RetryAfter string
	// Body is the optional response body.
	Body string
	// Headers contains additional response headers to set.
	Headers map[string]string
}

// route holds a path-specific response sequence and its request counter.
type route struct {
	mu           sync.Mutex
	responses    []Response
	requestCount int
	fallback     Response
}

// next returns the next response from the sequence, or the fallback.
func (r *route) next() Response {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requestCount++
	idx := r.requestCount - 1
	if idx < len(r.responses) {
		return r.responses[idx]
	}
	return r.fallback
}

// count returns the total number of requests handled by this route.
func (r *route) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.requestCount
}

// Server is a configurable mock HTTP server that responds with predefined
// sequences of HTTP responses to simulate real-world retry scenarios.
type Server struct {
	srv          *httptest.Server
	defaultRoute *route
	routes       map[string]*route
	mu           sync.RWMutex
}

// Option is a functional option for configuring a Server.
type Option func(*Server)

// WithFallback sets the default response used after the global sequence is
// exhausted. Defaults to 200 OK if not specified.
func WithFallback(r Response) Option {
	return func(s *Server) {
		s.defaultRoute.fallback = r
	}
}

// WithRoute registers a path-specific response sequence. Requests matching
// the path exactly will use this sequence instead of the default one.
// The fallback for the route defaults to 200 OK.
func WithRoute(path string, responses []Response, opts ...RouteOption) Option {
	return func(s *Server) {
		r := &route{
			responses: responses,
			fallback:  Response{StatusCode: http.StatusOK},
		}
		for _, opt := range opts {
			opt(r)
		}
		s.mu.Lock()
		s.routes[path] = r
		s.mu.Unlock()
	}
}

// RouteOption is a functional option for configuring an individual route.
type RouteOption func(*route)

// WithRouteFallback sets the fallback response for a specific route after
// its sequence is exhausted.
func WithRouteFallback(resp Response) RouteOption {
	return func(r *route) {
		r.fallback = resp
	}
}

// New creates and starts a mock HTTP server with the given default response
// sequence. The server is automatically stopped via t.Cleanup when the test
// ends. Pass nil or an empty slice for responses to use only the fallback.
func New(t *testing.T, responses []Response, opts ...Option) *Server {
	t.Helper()

	s := &Server{
		defaultRoute: &route{
			responses: responses,
			fallback:  Response{StatusCode: http.StatusOK},
		},
		routes: make(map[string]*route),
	}

	for _, opt := range opts {
		opt(s)
	}

	s.srv = httptest.NewServer(http.HandlerFunc(s.handler))
	t.Cleanup(s.srv.Close)

	return s
}

// URL returns the base URL of the mock server (e.g. "http://127.0.0.1:PORT").
func (s *Server) URL() string {
	return s.srv.URL
}

// RequestCount returns the total number of requests received by the default
// (catch-all) route. For path-specific counts use RouteRequestCount.
func (s *Server) RequestCount() int {
	return s.defaultRoute.count()
}

// RouteRequestCount returns the number of requests received by the given path.
// Returns 0 if the path was never registered or never received requests.
func (s *Server) RouteRequestCount(path string) int {
	s.mu.RLock()
	r, ok := s.routes[path]
	s.mu.RUnlock()
	if !ok {
		return 0
	}
	return r.count()
}

// Reset resets the request counters and sequence position for all routes,
// allowing the same server to be reused across sub-tests.
func (s *Server) Reset() {
	s.defaultRoute.mu.Lock()
	s.defaultRoute.requestCount = 0
	s.defaultRoute.mu.Unlock()

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, r := range s.routes {
		r.mu.Lock()
		r.requestCount = 0
		r.mu.Unlock()
	}
}

// handler dispatches incoming requests to the matching route or the default.
func (s *Server) handler(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	rt, ok := s.routes[r.URL.Path]
	s.mu.RUnlock()

	if !ok {
		rt = s.defaultRoute
	}

	resp := rt.next()
	writeResponse(w, resp)
}

// writeResponse writes a Response to the http.ResponseWriter.
func writeResponse(w http.ResponseWriter, resp Response) {
	for k, v := range resp.Headers {
		w.Header().Set(k, v)
	}
	if resp.RetryAfter != "" {
		w.Header().Set("Retry-After", resp.RetryAfter)
	}

	code := resp.StatusCode
	if code == 0 {
		code = http.StatusOK
	}
	w.WriteHeader(code)

	if resp.Body != "" {
		_, _ = w.Write([]byte(resp.Body))
	}
}
