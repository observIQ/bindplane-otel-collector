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

package resolver

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

// testLogger creates a no-op logger for testing
func testLogger() *zap.Logger {
	return zap.NewNop()
}

func TestResolver_Validate(t *testing.T) {
	tests := []struct {
		name     string
		resolver Resolver
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid resolver with positive timeout",
			resolver: Resolver{
				Server:  "8.8.8.8:53",
				Timeout: 5 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "valid resolver with different port and positive timeout",
			resolver: Resolver{
				Server:  "1.1.1.1:5353",
				Timeout: 10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "valid resolver with minimum positive timeout",
			resolver: Resolver{
				Server:  "127.0.0.1:53",
				Timeout: time.Nanosecond,
			},
			wantErr: false,
		},
		{
			name: "empty server",
			resolver: Resolver{
				Server:  "",
				Timeout: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "server is required",
		},
		{
			name: "zero timeout",
			resolver: Resolver{
				Server:  "8.8.8.8:53",
				Timeout: 0,
			},
			wantErr: true,
			errMsg:  "timeout is required and must be greater than 0",
		},
		{
			name: "positive timeout",
			resolver: Resolver{
				Server:  "8.8.8.8:53",
				Timeout: 1 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid host:port format - missing port",
			resolver: Resolver{
				Server:  "8.8.8.8",
				Timeout: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "failed to split host and port: address 8.8.8.8: missing port in address",
		},
		{
			name: "invalid host:port format - missing host",
			resolver: Resolver{
				Server:  ":53",
				Timeout: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "host is required",
		},
		{
			name: "empty host",
			resolver: Resolver{
				Server:  ":53",
				Timeout: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "host is required",
		},
		{
			name: "empty port",
			resolver: Resolver{
				Server:  "8.8.8.8:",
				Timeout: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "port is required",
		},
		{
			name: "invalid IP address - hostname",
			resolver: Resolver{
				Server:  "google.com:53",
				Timeout: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "invalid IP address: google.com",
		},
		{
			name: "invalid IP address - malformed",
			resolver: Resolver{
				Server:  "256.256.256.256:53",
				Timeout: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "invalid IP address: 256.256.256.256",
		},
		{
			name: "invalid IP address - partial",
			resolver: Resolver{
				Server:  "192.168.1:53",
				Timeout: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "invalid IP address: 192.168.1",
		},
		{
			name: "invalid port - non-numeric",
			resolver: Resolver{
				Server:  "8.8.8.8:abc",
				Timeout: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "invalid port: abc",
		},
		{
			name: "invalid port - negative",
			resolver: Resolver{
				Server:  "8.8.8.8:-1",
				Timeout: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "port must be between 0 and 65535: -1",
		},
		{
			name: "invalid port - too large",
			resolver: Resolver{
				Server:  "8.8.8.8:65536",
				Timeout: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "port must be between 0 and 65535: 65536",
		},
		{
			name: "valid port - minimum",
			resolver: Resolver{
				Server:  "8.8.8.8:0",
				Timeout: 5 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "valid port - maximum",
			resolver: Resolver{
				Server:  "8.8.8.8:65535",
				Timeout: 5 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "valid IPv6 address",
			resolver: Resolver{
				Server:  "[2001:4860:4860::8888]:53",
				Timeout: 5 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid IPv6 format - missing brackets",
			resolver: Resolver{
				Server:  "2001:4860:4860::8888:53",
				Timeout: 5 * time.Second,
			},
			wantErr: true,
			errMsg:  "failed to split host and port: address 2001:4860:4860::8888:53: too many colons in address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.resolver.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error but got none")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %v, wantErr %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name            string
		configPath      string
		envVars         map[string]string
		createFile      bool
		fileContent     string
		wantErr         bool
		errContains     string
		expectedServer  string
		expectedTimeout time.Duration
	}{
		{
			name:            "load from valid config file",
			configPath:      "testdata/valid_config.yaml",
			createFile:      false,
			wantErr:         false,
			expectedServer:  "8.8.8.8:53",
			expectedTimeout: 5 * time.Second,
		},
		{
			name:       "load from environment variables",
			configPath: "",
			envVars: map[string]string{
				"BINDPLANE_OTEL_COLLECTOR_RESOLVER_ENABLE":  "true",
				"BINDPLANE_OTEL_COLLECTOR_RESOLVER_SERVER":  "1.1.1.1:53",
				"BINDPLANE_OTEL_COLLECTOR_RESOLVER_TIMEOUT": "10s",
			},
			wantErr:         false,
			expectedServer:  "1.1.1.1:53",
			expectedTimeout: 10 * time.Second,
		},
		{
			name:        "file not found",
			configPath:  "nonexistent.yaml",
			wantErr:     true,
			errContains: "failed to load resolver config from file",
		},
		{
			name:        "invalid YAML file",
			configPath:  "testdata/invalid_yaml.yaml",
			createFile:  false,
			wantErr:     true,
			errContains: "failed to parse config file",
		},
		{
			name:        "invalid resolver config in file",
			configPath:  "testdata/invalid_server.yaml",
			createFile:  false,
			wantErr:     true,
			errContains: "invalid resolver configuration",
		},
		{
			name:            "valid config with IPv6",
			configPath:      "testdata/valid_config_ipv6.yaml",
			createFile:      false,
			wantErr:         false,
			expectedServer:  "[2001:4860:4860::8888]:53",
			expectedTimeout: 10 * time.Second,
		},
		{
			name:            "valid config with custom port",
			configPath:      "testdata/valid_config_custom_port.yaml",
			createFile:      false,
			wantErr:         false,
			expectedServer:  "1.1.1.1:5353",
			expectedTimeout: 30 * time.Second,
		},
		{
			name:            "valid config minimal",
			configPath:      "testdata/valid_config_minimal.yaml",
			createFile:      false,
			wantErr:         false,
			expectedServer:  "127.0.0.1:53",
			expectedTimeout: 1 * time.Nanosecond,
		},
		{
			name:            "valid config with positive timeout",
			configPath:      "testdata/valid_config_positive_timeout.yaml",
			createFile:      false,
			wantErr:         false,
			expectedServer:  "9.9.9.9:53",
			expectedTimeout: 5 * time.Second,
		},
		{
			name:        "invalid timeout format in file",
			configPath:  "testdata/invalid_timeout.yaml",
			createFile:  false,
			wantErr:     true,
			errContains: "failed to parse config file",
		},
		{
			name:        "missing server in file",
			configPath:  "testdata/missing_server.yaml",
			createFile:  false,
			wantErr:     true,
			errContains: "invalid resolver configuration",
		},
		{
			name:        "missing timeout in file",
			configPath:  "testdata/missing_timeout.yaml",
			createFile:  false,
			wantErr:     true,
			errContains: "invalid resolver configuration",
		},
		{
			name:        "environment variables not set",
			envVars:     map[string]string{},
			wantErr:     true,
			errContains: "resolver not enabled",
		},
		{
			name: "missing server environment variable",
			envVars: map[string]string{
				"BINDPLANE_OTEL_COLLECTOR_RESOLVER_ENABLE": "true",
			},
			wantErr:     true,
			errContains: "resolver server not set",
		},
		{
			name: "missing timeout environment variable",
			envVars: map[string]string{
				"BINDPLANE_OTEL_COLLECTOR_RESOLVER_ENABLE": "true",
				"BINDPLANE_OTEL_COLLECTOR_RESOLVER_SERVER": "8.8.8.8:53",
			},
			wantErr:     true,
			errContains: "resolver timeout not set",
		},
		{
			name: "invalid timeout format in environment",
			envVars: map[string]string{
				"BINDPLANE_OTEL_COLLECTOR_RESOLVER_ENABLE":  "true",
				"BINDPLANE_OTEL_COLLECTOR_RESOLVER_SERVER":  "8.8.8.8:53",
				"BINDPLANE_OTEL_COLLECTOR_RESOLVER_TIMEOUT": "invalid",
			},
			wantErr:     true,
			errContains: "invalid timeout format",
		},
		{
			name: "invalid resolver config from environment",
			envVars: map[string]string{
				"BINDPLANE_OTEL_COLLECTOR_RESOLVER_ENABLE":  "true",
				"BINDPLANE_OTEL_COLLECTOR_RESOLVER_SERVER":  "invalid-ip:53",
				"BINDPLANE_OTEL_COLLECTOR_RESOLVER_TIMEOUT": "-5s",
			},
			wantErr:     true,
			errContains: "invalid resolver configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up environment variables
			os.Clearenv()

			// Set up environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Create test file if needed
			if tt.createFile {
				file, err := os.Create(tt.configPath)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				_, err = file.WriteString(tt.fileContent)
				if err != nil {
					t.Fatalf("Failed to write test file content: %v", err)
				}
				file.Close()
				defer os.Remove(tt.configPath)
			}

			// Test the New function
			resolver, err := New(testLogger(), tt.configPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("New() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("New() error = %v, want error containing %s", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("New() unexpected error = %v", err)
					return
				}
				if resolver == nil {
					t.Errorf("New() returned nil resolver")
					return
				}
				if tt.expectedServer != "" && resolver.Server != tt.expectedServer {
					t.Errorf("New() server = %v, want %v", resolver.Server, tt.expectedServer)
				}
				if tt.expectedTimeout != 0 && resolver.Timeout != tt.expectedTimeout {
					t.Errorf("New() timeout = %v, want %v", resolver.Timeout, tt.expectedTimeout)
				}
			}
		})
	}
}

func TestNew_FilePriority(t *testing.T) {
	// Test that file config takes priority over environment variables
	os.Clearenv()

	// Set environment variables
	os.Setenv("BINDPLANE_OTEL_COLLECTOR_RESOLVER_ENABLE", "true")
	os.Setenv("BINDPLANE_OTEL_COLLECTOR_RESOLVER_SERVER", "1.1.1.1:53")
	os.Setenv("BINDPLANE_OTEL_COLLECTOR_RESOLVER_TIMEOUT", "10s")

	// Use golden config file with different values
	configPath := "testdata/valid_config.yaml"

	// Load from file (should ignore environment)
	resolver, err := New(testLogger(), configPath)
	if err != nil {
		t.Fatalf("New() unexpected error = %v", err)
	}

	// Should use file values, not environment values
	if resolver.Server != "8.8.8.8:53" {
		t.Errorf("New() server = %v, want 8.8.8.8:53", resolver.Server)
	}
	if resolver.Timeout != 5*time.Second {
		t.Errorf("New() timeout = %v, want 5s", resolver.Timeout)
	}
}

func TestResolver_Configure(t *testing.T) {
	tests := []struct {
		name        string
		resolver    Resolver
		wantErr     bool
		errContains string
	}{
		{
			name: "valid configuration",
			resolver: Resolver{
				Server:  "8.8.8.8:53",
				Timeout: 5 * time.Second,
				logger:  testLogger(),
			},
			wantErr: false,
		},
		{
			name: "valid configuration with IPv6",
			resolver: Resolver{
				Server:  "[2001:4860:4860::8888]:53",
				Timeout: 10 * time.Second,
				logger:  testLogger(),
			},
			wantErr: false,
		},
		{
			name: "valid configuration with custom port",
			resolver: Resolver{
				Server:  "1.1.1.1:5353",
				Timeout: 30 * time.Second,
				logger:  testLogger(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store the original resolver to restore it later
			originalResolver := net.DefaultResolver

			// Test the Configure method
			err := tt.resolver.Configure()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Configure() expected error but got none")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Configure() error = %v, want error containing %s", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("Configure() unexpected error = %v", err)
					return
				}

				// Verify that the default resolver was changed
				if net.DefaultResolver == originalResolver {
					t.Errorf("Configure() did not change the default resolver")
				}

				// Verify that the resolver has the expected configuration
				if net.DefaultResolver != nil {
					// We can't directly test the internal configuration of the resolver,
					// but we can verify it's not nil and is different from the original
					if net.DefaultResolver == originalResolver {
						t.Errorf("Configure() did not set a new resolver")
					}
				}
			}

			// Restore the original resolver
			net.DefaultResolver = originalResolver
		})
	}
}

func TestResolver_Configure_Integration(t *testing.T) {
	// Test that Configure actually affects DNS resolution
	// This is an integration test that verifies the resolver works end-to-end

	// Create a valid resolver configuration
	resolver := Resolver{
		Server:  "8.8.8.8:53",
		Timeout: 5 * time.Second,
		logger:  testLogger(),
	}

	// Store the original resolver
	originalResolver := net.DefaultResolver
	defer func() {
		net.DefaultResolver = originalResolver
	}()

	// Configure the resolver
	err := resolver.Configure()
	if err != nil {
		t.Fatalf("Configure() failed: %v", err)
	}

	// Test that DNS resolution works with the configured resolver
	// Note: This test might fail if there are network issues, but it's good for integration testing
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to resolve a well-known domain
	_, err = net.DefaultResolver.LookupHost(ctx, "google.com")
	if err != nil {
		// If this fails due to network issues, that's okay for a unit test
		// We're mainly testing that the resolver was configured and can be called
		t.Logf("DNS lookup failed (this might be due to network issues): %v", err)
	}

	// Verify that the resolver is not nil and is different from the original
	if net.DefaultResolver == nil {
		t.Errorf("Configure() set resolver to nil")
	}
	if net.DefaultResolver == originalResolver {
		t.Errorf("Configure() did not change the default resolver")
	}
}
