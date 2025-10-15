package resolver

import (
	"os"
	"testing"
	"time"
)

func TestLoadFromFile(t *testing.T) {
	tests := []struct {
		name            string
		configPath      string
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
			name:        "file not found",
			configPath:  "nonexistent.yaml",
			wantErr:     true,
			errContains: "failed to open config file",
		},
		{
			name:        "invalid YAML file",
			configPath:  "testdata/invalid_yaml.yaml",
			createFile:  false,
			wantErr:     true,
			errContains: "failed to parse config file",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file if needed
			if tt.createFile {
				err := os.WriteFile(tt.configPath, []byte(tt.fileContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				defer os.Remove(tt.configPath)
			}

			resolver, err := loadFromFile(tt.configPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("loadFromFile() expected error but got none")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("loadFromFile() error = %v, want error containing %s", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("loadFromFile() unexpected error = %v", err)
				return
			}

			if resolver.Server != tt.expectedServer {
				t.Errorf("loadFromFile() server = %v, want %v", resolver.Server, tt.expectedServer)
			}

			if resolver.Timeout != tt.expectedTimeout {
				t.Errorf("loadFromFile() timeout = %v, want %v", resolver.Timeout, tt.expectedTimeout)
			}
		})
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	tests := []struct {
		name            string
		envVars         map[string]string
		wantErr         bool
		errContains     string
		expectedServer  string
		expectedTimeout time.Duration
	}{
		{
			name: "load from environment variables",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Clearenv()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			resolver, err := loadFromEnvironment()

			if tt.wantErr {
				if err == nil {
					t.Errorf("loadFromEnvironment() expected error but got none")
					return
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("loadFromEnvironment() error = %v, want error containing %s", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("loadFromEnvironment() unexpected error = %v", err)
				return
			}

			if resolver.Server != tt.expectedServer {
				t.Errorf("loadFromEnvironment() server = %v, want %v", resolver.Server, tt.expectedServer)
			}

			if resolver.Timeout != tt.expectedTimeout {
				t.Errorf("loadFromEnvironment() timeout = %v, want %v", resolver.Timeout, tt.expectedTimeout)
			}
		})
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

// containsSubstring is a helper function for contains
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
