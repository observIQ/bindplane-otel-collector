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

package httpevents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/open-telemetry/opentelemetry-log-collection/testutil"
	"github.com/stretchr/testify/require"
)

const (
	authHeader        = "x-secret-key"
	localHost         = "localhost"
	defaultListenAddr = "localhost:0"
	devUser           = "dev"
)

func TestStartStop(t *testing.T) {
	address := localHost
	port := freeLocalHostPort()
	if port == 0 {
		t.Errorf("failed to find available port for test server")
		return
	}

	cfg := NewHTTPInputConfig("test_id")
	cfg.ListenAddress = fmt.Sprintf("%s:%d", address, port)
	op, err := cfg.build(testutil.NewBuildContext(t))
	require.NoError(t, err)
	require.NoError(t, op.Start(nil), "failed to start operator")
	require.NoError(t, op.Stop(), "failed to stop operator")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	op.goListen(ctx)

	// stopping again should not panic
	p := func() {
		err := op.Stop()
		require.NoError(t, err)
	}
	require.NotPanics(t, p)
}

func TestServer(t *testing.T) {
	address := localHost
	port := freeLocalHostPort()
	if port == 0 {
		t.Errorf("failed to find available port for test server")
		return
	}

	cfg := NewHTTPInputConfig("test_id")
	cfg.ListenAddress = fmt.Sprintf("%s:%d", address, port)
	cfg.MaxBodySize = 50
	op, err := cfg.build(testutil.NewBuildContext(t))
	if err != nil {
		require.NoError(t, err)
		return
	}
	if err := op.Start(nil); err != nil {
		require.NoError(t, err)
	}
	defer func() {
		if err := op.Stop(); err != nil {
			t.Errorf(err.Error())
		}
	}()

	require.NoError(t, testConnection(cfg.ListenAddress), "expected http server to start and accept requests")

	cases := []struct {
		name         string
		inputRequest *http.Request
		expectStatus int
	}{
		{
			"basic-event",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/",
				}

				raw := map[string]interface{}{
					"message": "this is a basic event",
				}
				b, _ := json.Marshal(raw)
				buf := bytes.NewBuffer(b)

				req, _ := http.NewRequest("POST", u.String(), buf)
				return req
			}(),
			201,
		},
		{
			"health",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/health",
				}

				req, _ := http.NewRequest("GET", u.String(), nil)
				return req
			}(),
			200,
		},
		{
			"invalid-json-request",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/",
				}

				b, _ := json.Marshal([]byte(`some string`))
				buf := bytes.NewBuffer(b)

				req, _ := http.NewRequest("POST", u.String(), buf)
				return req
			}(),
			400,
		},
		{
			"request-to-large",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/",
				}

				raw := map[string]interface{}{
					"message":     "this is a basic event",
					"large_field": "this is a large field that will cause a body to large error",
				}
				b, _ := json.Marshal(raw)
				buf := bytes.NewBuffer(b)

				req, err := http.NewRequest("POST", u.String(), buf)
				require.NoError(t, err)
				defer req.Body.Close()
				return req
			}(),
			413,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := http.Client{}

			resp, err := client.Do(tc.inputRequest)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, tc.expectStatus, resp.StatusCode)
		})
	}
}

func TestServerBasicAuth(t *testing.T) {
	address := localHost
	port := freeLocalHostPort()
	if port == 0 {
		t.Errorf("failed to find available port for test server")
		return
	}

	cfg := NewHTTPInputConfig("test_id")
	cfg.ListenAddress = fmt.Sprintf("%s:%d", address, port)
	cfg.AuthConfig.Username = "stanza"
	cfg.AuthConfig.Password = "dev"
	op, err := cfg.build(testutil.NewBuildContext(t))
	if err != nil {
		require.NoError(t, err)
		return
	}
	if err := op.Start(nil); err != nil {
		require.NoError(t, err)
	}
	defer func() {
		if err := op.Stop(); err != nil {
			t.Errorf(err.Error())
		}
	}()

	require.NoError(t, testConnection(cfg.ListenAddress), "expected http server to start and accept requests")

	cases := []struct {
		name         string
		inputRequest *http.Request
		expectStatus int
	}{
		{
			"missing-auth",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/",
				}

				raw := map[string]interface{}{
					"message": "this is a basic event",
				}
				b, _ := json.Marshal(raw)
				buf := bytes.NewBuffer(b)

				req, _ := http.NewRequest("POST", u.String(), buf)
				return req
			}(),
			403,
		},
		{
			"auth-not-required",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/",
				}
				req, _ := http.NewRequest("GET", u.String(), nil)
				return req
			}(),
			200,
		},
		{
			"auth-required",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/",
				}
				req, _ := http.NewRequest("POST", u.String(), nil)
				return req
			}(),
			403,
		},
		{
			"auth-required-get",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/",
				}

				raw := map[string]interface{}{
					"message": "this is a basic event",
				}
				b, _ := json.Marshal(raw)
				buf := bytes.NewBuffer(b)

				req, _ := http.NewRequest("GET", u.String(), buf)
				return req
			}(),
			403,
		},
		{
			"auth-not-required-health",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/",
				}
				req, _ := http.NewRequest("GET", u.String(), nil)
				return req
			}(),
			200,
		},
		{
			"valid",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/",
				}

				raw := map[string]interface{}{
					"message": "this is a basic event",
				}
				b, _ := json.Marshal(raw)
				buf := bytes.NewBuffer(b)

				req, _ := http.NewRequest("POST", u.String(), buf)
				req.SetBasicAuth("stanza", "dev")
				return req
			}(),
			201,
		},
		{
			"invalid-password",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/",
				}

				raw := map[string]interface{}{
					"message": "this is a basic event",
				}
				b, _ := json.Marshal(raw)
				buf := bytes.NewBuffer(b)

				req, _ := http.NewRequest("POST", u.String(), buf)
				req.SetBasicAuth("stanza", "bad-password")
				return req
			}(),
			403,
		},
		{
			"invalid-username",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/",
				}

				raw := map[string]interface{}{
					"message": "this is a basic event",
				}
				b, _ := json.Marshal(raw)
				buf := bytes.NewBuffer(b)

				req, _ := http.NewRequest("POST", u.String(), buf)
				req.SetBasicAuth("wrong-username", "dev")
				return req
			}(),
			403,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := http.Client{}

			resp, err := client.Do(tc.inputRequest)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, tc.expectStatus, resp.StatusCode)
		})
	}
}

func TestServerTokenAuth(t *testing.T) {
	address := localHost
	port := freeLocalHostPort()
	if port == 0 {
		t.Errorf("failed to find available port for test server")
		return
	}

	cfg := NewHTTPInputConfig("test_id")
	cfg.ListenAddress = fmt.Sprintf("%s:%d", address, port)
	cfg.AuthConfig.TokenHeader = "x-secret-key"
	cfg.AuthConfig.Tokens = []string{"test-token", "test-token-2"}
	op, err := cfg.build(testutil.NewBuildContext(t))
	if err != nil {
		require.NoError(t, err)
		return
	}
	if err := op.Start(nil); err != nil {
		require.NoError(t, err)
	}
	defer func() {
		if err := op.Stop(); err != nil {
			t.Errorf(err.Error())
		}
	}()

	require.NoError(t, testConnection(cfg.ListenAddress), "expected http server to start and accept requests")

	cases := []struct {
		name         string
		inputRequest *http.Request
		expectStatus int
	}{
		{
			"test-token",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/",
				}

				raw := map[string]interface{}{
					"message": "this is a basic event",
				}
				b, _ := json.Marshal(raw)
				buf := bytes.NewBuffer(b)

				req, _ := http.NewRequest("POST", u.String(), buf)
				req.Header["x-secret-key"] = []string{"test-token"}
				return req
			}(),
			201,
		},
		{
			"auth-not-required",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/",
				}
				req, _ := http.NewRequest("GET", u.String(), nil)
				return req
			}(),
			200,
		},
		{
			"auth-not-required-health",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/health",
				}
				req, _ := http.NewRequest("GET", u.String(), nil)
				return req
			}(),
			200,
		},
		{
			"auth-required",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/",
				}
				req, _ := http.NewRequest("POST", u.String(), nil)
				return req
			}(),
			403,
		},
		{
			"auth-required-get",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/",
				}

				raw := map[string]interface{}{
					"message": "this is a basic event",
				}
				b, _ := json.Marshal(raw)
				buf := bytes.NewBuffer(b)

				req, _ := http.NewRequest("GET", u.String(), buf)
				return req
			}(),
			403,
		},
		{
			"test-token2",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/",
				}

				raw := map[string]interface{}{
					"message": "this is a basic event",
				}
				b, _ := json.Marshal(raw)
				buf := bytes.NewBuffer(b)

				req, _ := http.NewRequest("POST", u.String(), buf)
				req.Header["x-secret-key"] = []string{"test-token"}
				return req
			}(),
			201,
		},
		{
			"invalid-token",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/",
				}

				raw := map[string]interface{}{
					"message": "this is a basic event",
				}
				b, _ := json.Marshal(raw)
				buf := bytes.NewBuffer(b)

				req, _ := http.NewRequest("POST", u.String(), buf)
				req.Header["x-secret-key"] = []string{"invalid"}
				return req
			}(),
			403,
		},
		{
			"invalid-header",
			func() *http.Request {
				u := url.URL{
					Scheme: "http",
					Host:   cfg.ListenAddress,
					Path:   "/",
				}

				raw := map[string]interface{}{
					"message": "this is a basic event",
				}
				b, _ := json.Marshal(raw)
				buf := bytes.NewBuffer(b)

				req, _ := http.NewRequest("POST", u.String(), buf)
				req.Header["x-invalid-key"] = []string{"test-token"}
				return req
			}(),
			403,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client := http.Client{}

			resp, err := client.Do(tc.inputRequest)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, tc.expectStatus, resp.StatusCode)
		})
	}
}

func TestParse(t *testing.T) {
	cases := []struct {
		name         string
		payload      map[string]interface{}
		req          *http.Request
		expect       *entry.Entry
		expectErr    bool
		expectErrStr string
	}{
		{
			"nil-payload",
			nil,
			&http.Request{},
			nil,
			true,
			"payload and http request must be set",
		},
		{
			"nil-request",
			make(map[string]interface{}),
			nil,
			nil,
			true,
			"payload and http request must be set",
		},
		{
			"valid-request",
			map[string]interface{}{
				"message": "generic event",
			},
			&http.Request{
				RemoteAddr: "10.1.1.1:5555",
				Host:       "1.1.1.1:80",
				Proto:      "HTTP/1.1",
			},
			&entry.Entry{
				Body: map[string]interface{}{
					"message": "generic event",
				},
				Attributes: map[string]string{
					"net.peer.ip":      "10.1.1.1",
					"net.peer.port":    "5555",
					"net.host.ip":      "1.1.1.1",
					"net.host.port":    "80",
					"protocol":         "HTTP",
					"protocol_version": "1.1",
				},
			},
			false,
			"",
		}, {
			"valid-request-without-message",
			map[string]interface{}{
				"msg":   "generic event",
				"stage": "dev",
			},
			&http.Request{
				RemoteAddr: "10.1.1.1:5555",
				Host:       "1.1.1.1:80",
				Proto:      "HTTP/1.1",
			},
			&entry.Entry{
				Body: map[string]interface{}{
					"msg":   "generic event",
					"stage": "dev",
				},
				Attributes: map[string]string{
					"net.peer.ip":      "10.1.1.1",
					"net.peer.port":    "5555",
					"net.host.ip":      "1.1.1.1",
					"net.host.port":    "80",
					"protocol":         "HTTP",
					"protocol_version": "1.1",
				},
			},
			false,
			"",
		},
		{
			"large-request",
			map[string]interface{}{
				"message":  "generic event",
				"event_id": 155,
				"dev_mode": true,
				"params": map[string]string{
					"mode": "cluster",
					"user": "admin",
				},
			},
			&http.Request{
				RemoteAddr: "10.1.1.1:5555",
				Host:       "1.1.1.1:80",
				Proto:      "HTTP/1.1",
			},
			&entry.Entry{
				Body: map[string]interface{}{
					"message":  "generic event",
					"event_id": 155,
					"dev_mode": true,
					"params": map[string]string{
						"mode": "cluster",
						"user": "admin",
					},
				},
				Attributes: map[string]string{
					"net.peer.ip":      "10.1.1.1",
					"net.peer.port":    "5555",
					"net.host.ip":      "1.1.1.1",
					"net.host.port":    "80",
					"protocol":         "HTTP",
					"protocol_version": "1.1",
				},
			},
			false,
			"",
		},
		{
			"invalid-peer-addr",
			map[string]interface{}{
				"message": "generic event",
			},
			&http.Request{
				RemoteAddr: "10.1.1.1", // should not be set in entry labels
				Host:       "1.1.1.1:80",
				Proto:      "HTTP/1.1",
			},
			&entry.Entry{
				Body: map[string]interface{}{
					"message": "generic event",
				},
				Attributes: map[string]string{
					"net.host.ip":      "1.1.1.1",
					"net.host.port":    "80",
					"protocol":         "HTTP",
					"protocol_version": "1.1",
				},
			},
			false,
			"",
		},
		{
			"invalid-host-addr",
			map[string]interface{}{
				"message": "generic event",
			},
			&http.Request{
				RemoteAddr: "10.1.1.1:5555",
				Host:       "1.1.1.1",
				Proto:      "HTTP/1.1",
			},
			&entry.Entry{
				Body: map[string]interface{}{
					"message": "generic event",
				},
				Attributes: map[string]string{
					"net.peer.ip":      "10.1.1.1",
					"net.peer.port":    "5555",
					"protocol":         "HTTP",
					"protocol_version": "1.1",
				},
			},
			false,
			"",
		},
		{
			"invalid-proto",
			map[string]interface{}{
				"message": "generic event",
			},
			&http.Request{
				RemoteAddr: "10.1.1.1:5555",
				Host:       "1.1.1.1:80",
				Proto:      "HTTP",
			},
			&entry.Entry{
				Body: map[string]interface{}{
					"message": "generic event",
				},
				Attributes: map[string]string{
					"net.peer.ip":   "10.1.1.1",
					"net.peer.port": "5555",
					"net.host.ip":   "1.1.1.1",
					"net.host.port": "80",
				},
			},
			false,
			"",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := NewHTTPInputConfig("test_id")
			cfg.ListenAddress = ":0"
			op, err := cfg.build(testutil.NewBuildContext(t))
			if err != nil {
				require.NoError(t, err)
				return
			}

			e, err := op.parse(tc.payload, tc.req)
			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectErrStr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, e)
			require.Equal(t, tc.expect.Body, e.Body)
			require.Equal(t, tc.expect.Attributes, e.Attributes)
			require.Equal(t, tc.expect.Resource, e.Resource)
			require.NotZero(t, e.Timestamp)
		})
	}
}

func TestAddPeerLabelsError(t *testing.T) {
	e := entry.New()
	// ip without port
	require.Error(t, addPeerLabels("127.0.0.1", e))
	// port without ip
	require.Error(t, addPeerLabels("443", e))
}

func TestAddHostLabelsError(t *testing.T) {
	e := entry.New()
	// ip without port
	require.Error(t, addHostLabels("127.0.0.1", e))
	// port without ip
	require.Error(t, addHostLabels("443", e))
}

func TestAddProtoLabelsError(t *testing.T) {
	e := entry.New()
	require.Error(t, addProtoLabels("HTTP", e))
	require.Error(t, addProtoLabels("1.1", e))
	require.Error(t, addProtoLabels("HTTP/t", e))
}

func freeLocalHostPort() int {
	port := 0
	minPort := 40000
	maxPort := 50000
	for i := 1; i < 50; i++ {
		port = minPort + rand.Intn(maxPort-minPort+1)
		d, err := net.DialTimeout("tcp", net.JoinHostPort(localHost, strconv.Itoa(port)), time.Second*2)
		if err == nil {
			d.Close()
			break
		}
	}
	return port
}

func testConnection(address string) error {
	u := url.URL{
		Scheme: "http",
		Host:   address,
		Path:   "/health",
	}

	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: time.Second * 2,
	}

	attempt := 0
	for {
		resp, err := client.Do(req)
		// return when error is nil, connection
		// test passed
		if err == nil {
			return resp.Body.Close()
		}

		if attempt == 5 {
			return fmt.Errorf("test connection failed, the http server may not have started correctly: %s", err)
		}
		attempt++
		time.Sleep(time.Millisecond * 500)
	}
}
