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

package qradar

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net"
	"testing"
	"time"

	"github.com/observiq/bindplane-agent/exporter/qradar/internal/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter"
)

func Test_exporter_Capabilities(t *testing.T) {
	exp := &qradarExporter{}
	capabilities := exp.Capabilities()
	require.False(t, capabilities.MutatesData)
}

func TestLogDataPushingNetwork(t *testing.T) {
	// Channel to signal when log is received
	logReceived := make(chan bool)

	// Set up a mock Syslog server
	ln, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				conn, err := ln.Accept()
				if err != nil {
					log.Println("Error accepting connection:", err)
					return
				}
				go handleSyslogConnection(t, conn, logReceived)
			}
		}
	}()

	// Configure the exporter to use the mock Syslog server
	cfg := &Config{
		Syslog: SyslogConfig{
			AddrConfig: confignet.AddrConfig{
				Endpoint:  ln.Addr().String(),
				Transport: "tcp",
			},
		},
	}
	exporter, _ := newExporter(cfg, exporter.Settings{})

	// Mock log data
	ld := mockLogs(mockLogRecord(t, "test", map[string]any{"test": "test"}))

	// Test log data pushing
	err = exporter.logsDataPusher(context.Background(), ld)
	require.NoError(t, err)

	select {
	case <-logReceived:
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for log to be received")
	}
}

func handleSyslogConnection(t *testing.T, conn net.Conn, logReceived chan bool) {
	defer conn.Close()

	// Buffer to store the received data
	buf := make([]byte, 1024)

	// Read data from the connection
	n, err := conn.Read(buf)
	require.NoError(t, err)

	// Extract the received message
	receivedData := string(buf[:n])

	require.Equal(t, "{\"attributes\":{\"test\":\"test\"},\"body\":\"test\",\"resource_attributes\":{}}\n", receivedData)

	logReceived <- true
	conn.Close()
}

func TestOpenWriter(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*mocks.MockForwarderClient)
		expectedError bool
		cfg           Config
	}{

		{
			name: "Successful Syslog Open",
			setupMock: func(mockClient *mocks.MockForwarderClient) {
				mockClient.On("Dial", "tcp", "localhost:1234").Return(&net.TCPConn{}, nil)
			},
			expectedError: false,
			cfg: Config{
				Syslog: SyslogConfig{
					AddrConfig: confignet.AddrConfig{
						Endpoint:  "localhost:1234",
						Transport: "tcp",
					},
				},
			},
		},
		{
			name: "Syslog Open Error",
			setupMock: func(mockClient *mocks.MockForwarderClient) {
				mockClient.On("Dial", "tcp", "invalidendpoint").Return(nil, errors.New("error opening syslog"))
			},
			expectedError: true,
			cfg: Config{
				Syslog: SyslogConfig{
					AddrConfig: confignet.AddrConfig{
						Endpoint:  "invalidendpoint",
						Transport: "tcp",
					},
				},
			},
		},
		{
			name: "Successful TLS Dial",
			setupMock: func(mockClient *mocks.MockForwarderClient) {
				mockClient.On("DialWithTLS", "tcp", "localhost:1234", mock.Anything).Return(&tls.Conn{}, nil)
			},
			cfg: Config{
				Syslog: SyslogConfig{
					AddrConfig: confignet.AddrConfig{
						Endpoint:  "localhost:1234",
						Transport: "tcp",
					},
					TLSSetting: &configtls.ClientConfig{Insecure: true},
				},
			},
			expectedError: false,
		},
		{
			name: "Failed TLS Dial",
			setupMock: func(mockClient *mocks.MockForwarderClient) {
				mockClient.On("DialWithTLS", "tcp", "localhost:1234", mock.Anything).Return(nil, errors.New("TLS dial error"))
			},
			cfg: Config{
				Syslog: SyslogConfig{
					AddrConfig: confignet.AddrConfig{
						Endpoint:  "localhost:1234",
						Transport: "tcp",
					},
					TLSSetting: &configtls.ClientConfig{Insecure: true},
				},
			},
			expectedError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock client
			mockClient := mocks.NewMockForwarderClient(t)
			tc.setupMock(mockClient)

			// Create an instance of qradarExporter with the mock client
			exporter := &qradarExporter{
				qradarClient: mockClient,
				cfg:          &tc.cfg,
			}

			// Call openWriter
			_, err := exporter.openWriter(context.Background())

			// Assert the outcome
			if tc.expectedError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Assert mock interactions
			mockClient.AssertExpectations(t)
		})
	}
}
