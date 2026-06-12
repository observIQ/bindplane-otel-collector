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

package runtime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
	"github.com/observiq/bindplane-otel-collector/internal/extension/opampconnectionextension/internal/opamp"
	"gopkg.in/yaml.v3"
)

// Bindplane bootstrap env vars. When manager.yaml doesn't exist, these are
// the inputs the runtime uses to synthesize one (so a fresh agent can connect
// without needing a pre-written manager.yaml on disk).
const (
	endpointENV      = "OPAMP_ENDPOINT"
	agentIDENV       = "OPAMP_AGENT_ID"
	secretKeyENV     = "OPAMP_SECRET_KEY" //#nosec G101
	labelsENV        = "OPAMP_LABELS"
	agentNameENV     = "OPAMP_AGENT_NAME"
	tlsSkipVerifyENV = "OPAMP_TLS_SKIP_VERIFY"
	tlsCaENV         = "OPAMP_TLS_CA"
	tlsCertENV       = "OPAMP_TLS_CERT"
	tlsKeyENV        = "OPAMP_TLS_KEY"
)

// bootstrapManagerConfig returns nil if the agent should run in managed mode
// (either manager.yaml exists, or OPAMP_ENDPOINT is set and a new manager.yaml
// gets written from env vars). Returns os.ErrNotExist if neither path is true
// — caller drops to standalone mode.
func bootstrapManagerConfig(configPath *string) error {
	_, statErr := os.Stat(*configPath)
	switch {
	case statErr == nil:
		return ensureIdentity(*configPath)
	case errors.Is(statErr, os.ErrNotExist):
		newConfig := &opamp.Config{}

		var ok bool
		newConfig.Endpoint, ok = os.LookupEnv(endpointENV)
		if !ok {
			// No file, no env — standalone mode.
			return statErr
		}

		if envString, ok := os.LookupEnv(agentIDENV); ok {
			var err error
			newConfig.AgentID, err = opamp.ParseAgentID(envString)
			if err != nil {
				return fmt.Errorf("invalid agent ID in env %q: %w", agentIDENV, err)
			}
		} else {
			u, err := uuid.NewV7()
			if err != nil {
				return fmt.Errorf("new uuidv7: %w", err)
			}
			newConfig.AgentID = opamp.AgentIDFromUUID(u)
		}

		if sk, ok := os.LookupEnv(secretKeyENV); ok {
			newConfig.SecretKey = &sk
		}
		if an, ok := os.LookupEnv(agentNameENV); ok {
			newConfig.AgentName = &an
		}
		if label, ok := os.LookupEnv(labelsENV); ok {
			newConfig.Labels = &label
		}

		tlsConfig, err := tlsFromEnv()
		if err != nil {
			return fmt.Errorf("failed to configure tls: %w", err)
		}
		if tlsConfig != nil {
			newConfig.TLS = tlsConfig
		}

		data, err := yaml.Marshal(newConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		if err := os.WriteFile(*configPath, data, 0600); err != nil {
			return fmt.Errorf("failed to write config file created from ENVs: %w", err)
		}
		return nil
	}
	return statErr
}

// ensureIdentity stamps a generated AgentID into manager.yaml on disk if the
// file exists but has none. Prevents anonymous connection attempts.
func ensureIdentity(configPath string) error {
	cBytes, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		return fmt.Errorf("unable to read file: %w", err)
	}
	var candidateConfig opamp.Config
	if err := yaml.Unmarshal(cBytes, &candidateConfig); err != nil {
		return fmt.Errorf("unable to interpret config file: %w", err)
	}
	if candidateConfig.AgentID != opamp.EmptyAgentID {
		return nil
	}

	u, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("new uuidv7: %w", err)
	}
	candidateConfig.AgentID = opamp.AgentIDFromUUID(u)

	newBytes, err := yaml.Marshal(candidateConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal sanitized config: %w", err)
	}
	if err = os.WriteFile(filepath.Clean(configPath), newBytes, 0600); err != nil {
		return fmt.Errorf("failed to rewrite manager config with identifying fields: %w", err)
	}
	return nil
}

// checkForCollectorRollbackConfig promotes <config>.rollback back over the
// current collector config and removes the rollback marker. Runs once at
// startup before the OpAMP client connects; covers the case where a remote
// config push crashed the agent mid-restart.
func checkForCollectorRollbackConfig(configPath string) error {
	cleanPath := filepath.Clean(configPath)
	rollbackFileName := fmt.Sprintf("%s.rollback", cleanPath)

	if _, err := os.Stat(rollbackFileName); errors.Is(err, os.ErrNotExist) {
		return nil
	}

	//#nosec G304 -- path is cleaned above
	contents, err := os.ReadFile(rollbackFileName)
	if err != nil {
		return fmt.Errorf("error while reading in collector rollback file: %w", err)
	}
	if err := os.WriteFile(cleanPath, contents, 0600); err != nil {
		return fmt.Errorf("error while writing rollback contents onto config: %w", err)
	}
	if err := os.Remove(rollbackFileName); err != nil {
		return fmt.Errorf("error while cleaning up rollback file: %w", err)
	}
	return nil
}

// tlsFromEnv builds an opamp TLS struct from OPAMP_TLS_* env vars. Returns nil
// when no TLS env vars are set (the caller leaves TLS unconfigured).
func tlsFromEnv() (*opamp.TLSConfig, error) {
	tlsConfig := opamp.TLSConfig{}
	configured := false

	if skipVerify := os.Getenv(tlsSkipVerifyENV); skipVerify != "" {
		s, err := strconv.ParseBool(skipVerify)
		if err != nil {
			return nil, fmt.Errorf("invalid value '%s' for environment option '%s': %w", skipVerify, tlsSkipVerifyENV, err)
		}
		tlsConfig.InsecureSkipVerify = s
		configured = true
	}
	if ca := os.Getenv(tlsCaENV); ca != "" {
		tlsConfig.CAFile = &ca
		configured = true
	}
	if crt := os.Getenv(tlsCertENV); crt != "" {
		tlsConfig.CertFile = &crt
		configured = true
	}
	if key := os.Getenv(tlsKeyENV); key != "" {
		tlsConfig.KeyFile = &key
		configured = true
	}

	if configured {
		return &tlsConfig, nil
	}
	return nil, nil
}
