package resolver

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// loadFromFile loads resolver configuration from a YAML file
func loadFromFile(configPath string) (Resolver, error) {
	var resolver Resolver

	//#nosec G304 -- Resolver config is user provided and protected by a
	// buffered reader with 256 byte limit
	file, err := os.Open(configPath)
	if err != nil {
		return resolver, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, 256)
	data, err := io.ReadAll(reader)
	if err != nil {
		return resolver, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &resolver); err != nil {
		return resolver, fmt.Errorf("failed to parse config file: %w", err)
	}

	return resolver, nil
}

// loadFromEnvironment loads resolver configuration from environment variables
func loadFromEnvironment() (Resolver, error) {
	var resolver Resolver

	enable := os.Getenv(ENVEnable)
	if enable == "" {
		return resolver, errors.New("resolver not enabled - set BINDPLANE_OTEL_COLLECTOR_RESOLVER_ENABLE environment variable")
	}

	server := os.Getenv(ENVServer)
	if server == "" {
		return resolver, fmt.Errorf("resolver server not set - set %s environment variable", ENVServer)
	}
	resolver.Server = server

	timeoutStr := os.Getenv(ENVTimeout)
	if timeoutStr == "" {
		return resolver, fmt.Errorf("resolver timeout not set - set %s environment variable", ENVTimeout)
	}

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return resolver, fmt.Errorf("invalid timeout format: %w", err)
	}
	resolver.Timeout = timeout

	return resolver, nil
}
