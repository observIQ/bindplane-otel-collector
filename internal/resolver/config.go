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
	// buffered reader
	file, err := os.Open(configPath)
	if err != nil {
		return resolver, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	reader := io.LimitReader(file, 256)
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
