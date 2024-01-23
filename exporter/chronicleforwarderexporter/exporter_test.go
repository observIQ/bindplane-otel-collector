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

package chronicleforwarderexporter

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/observiq/bindplane-agent/exporter/chronicleforwarderexporter/internal/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter"
)

func Test_exporter_Capabilities(t *testing.T) {
	exp := &chronicleForwarderExporter{}
	capabilities := exp.Capabilities()
	require.False(t, capabilities.MutatesData)
}

func TestLogDataPushingFile(t *testing.T) {
	// Open a temporary file for testing
	f, err := os.CreateTemp("", "test")
	require.NoError(t, err)
	defer f.Close()
	defer os.Remove(f.Name()) // Clean up the file afterwards

	cfg := &Config{
		ExportType: exportTypeFile,
		File: File{
			Path: f.Name(),
		},
	}
	exporter, _ := newExporter(cfg, exporter.CreateSettings{})

	// Mock log data
	ld := mockLogs(mockLogRecord(t, "test", map[string]any{"test": "test"}))

	err = exporter.logsDataPusher(context.Background(), ld)
	require.NoError(t, err)

	// Read the contents of the file
	content, err := os.ReadFile(f.Name())
	require.NoError(t, err)

	// Convert the content to a string and compare with the expected output
	receivedData := string(content)
	expectedData := "{\"attributes\":{\"test\":\"test\"},\"body\":\"test\",\"resource_attributes\":{}}\n"
	require.Equal(t, expectedData, receivedData, "File content does not match expected output")
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
		ExportType: exportTypeSyslog,
		Syslog: SyslogConfig{
			NetAddr: confignet.NetAddr{
				Endpoint:  ln.Addr().String(),
				Transport: "tcp",
			},
		},
	}
	exporter, _ := newExporter(cfg, exporter.CreateSettings{})

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
			name: "Successful File Open",
			setupMock: func(mockClient *mocks.MockForwarderClient) {
				mockClient.On("OpenFile", "testfile.log").Return(&os.File{}, nil)
			},
			expectedError: false,
			cfg: Config{
				ExportType: exportTypeFile,
				File: File{
					Path: "testfile.log",
				},
			},
		},
		{
			name: "File Open Error",
			setupMock: func(mockClient *mocks.MockForwarderClient) {
				mockClient.On("OpenFile", "invalidfile.log").Return(nil, errors.New("error opening file"))
			},
			expectedError: true,
			cfg: Config{
				ExportType: exportTypeFile,
				File: File{
					Path: "invalidfile.log",
				},
			},
		},
		{
			name: "Successful Syslog Open",
			setupMock: func(mockClient *mocks.MockForwarderClient) {
				mockClient.On("Dial", "tcp", "localhost:1234").Return(&net.TCPConn{}, nil)
			},
			expectedError: false,
			cfg: Config{
				ExportType: exportTypeSyslog,
				Syslog: SyslogConfig{
					NetAddr: confignet.NetAddr{
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
				ExportType: exportTypeSyslog,
				Syslog: SyslogConfig{
					NetAddr: confignet.NetAddr{
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
				ExportType: exportTypeSyslog,
				Syslog: SyslogConfig{
					NetAddr: confignet.NetAddr{
						Endpoint:  "localhost:1234",
						Transport: "tcp",
					},
					TLSSetting: &configtls.TLSClientSetting{Insecure: true},
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
				ExportType: exportTypeSyslog,
				Syslog: SyslogConfig{
					NetAddr: confignet.NetAddr{
						Endpoint:  "localhost:1234",
						Transport: "tcp",
					},
					TLSSetting: &configtls.TLSClientSetting{Insecure: true},
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

			// Create an instance of chronicleForwarderExporter with the mock client
			exporter := &chronicleForwarderExporter{
				chronicleForwarderClient: mockClient,
				cfg:                      &tc.cfg,
			}

			// Call openWriter
			_, err := exporter.openWriter()

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
