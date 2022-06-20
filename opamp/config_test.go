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

package opamp

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	// Keep this outside so it can be referenced as pointer
	secretKeyContents := "b92222ee-a1fc-4bb1-98db-26de3448541b"
	labelsContents := "one=foo,two=bar"
	agentNameContents := "My Agent"

	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Failed File Read",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				cfg, err := ParseConfig(configPath)
				assert.ErrorContains(t, err, errPrefixReadFile)
				assert.Nil(t, cfg)
			},
		},
		{
			desc: "Failed Marshal",
			testFunc: func(t *testing.T) {
				configContents := `
				{
					"endpoint": "localhost:1234"
				}`

				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				err := os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				cfg, err := ParseConfig(configPath)
				assert.ErrorContains(t, err, errPrefixParse)
				assert.Nil(t, cfg)
			},
		},
		{
			desc: "Successful Full Parse",
			testFunc: func(t *testing.T) {
				configContents := `
endpoint: localhost:1234
secret_key: b92222ee-a1fc-4bb1-98db-26de3448541b
agent_id: 8321f735-a52c-4f49-aca9-66f9266c5fe5
labels: "one=foo,two=bar"
agent_name: "My Agent"
`

				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				err := os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				expectedConfig := &Config{
					Endpoint:  "localhost:1234",
					SecretKey: &secretKeyContents,
					AgentID:   "8321f735-a52c-4f49-aca9-66f9266c5fe5",
					Labels:    &labelsContents,
					AgentName: &agentNameContents,
				}

				cfg, err := ParseConfig(configPath)
				assert.NoError(t, err)
				assert.Equal(t, expectedConfig, cfg)
			},
		},
		{
			desc: "Successful Partial Parse",
			testFunc: func(t *testing.T) {
				configContents := `
endpoint: localhost:1234
agent_id: 8321f735-a52c-4f49-aca9-66f9266c5fe5
`

				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				err := os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				expectedConfig := &Config{
					Endpoint:  "localhost:1234",
					SecretKey: nil,
					AgentID:   "8321f735-a52c-4f49-aca9-66f9266c5fe5",
					Labels:    nil,
					AgentName: nil,
				}

				cfg, err := ParseConfig(configPath)
				assert.NoError(t, err)
				assert.Equal(t, expectedConfig, cfg)
			},
		},
		{
			desc: "Successful Full Parse with TLS Insecure",
			testFunc: func(t *testing.T) {
				configContents := `
endpoint: localhost:1234
secret_key: b92222ee-a1fc-4bb1-98db-26de3448541b
agent_id: 8321f735-a52c-4f49-aca9-66f9266c5fe5
labels: "one=foo,two=bar"
agent_name: "My Agent"
tls_config:
  insecure: true
`

				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				err := os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				expectedConfig := &Config{
					Endpoint:  "localhost:1234",
					SecretKey: &secretKeyContents,
					AgentID:   "8321f735-a52c-4f49-aca9-66f9266c5fe5",
					Labels:    &labelsContents,
					AgentName: &agentNameContents,
					TLS: &TLSConfig{
						Insecure: true,
					},
				}

				cfg, err := ParseConfig(configPath)
				assert.NoError(t, err)
				assert.Equal(t, expectedConfig, cfg)
			},
		},
		{
			desc: "Successful Full Parse with TLS Secure Root CA",
			testFunc: func(t *testing.T) {
				configContents := `
endpoint: localhost:1234
secret_key: b92222ee-a1fc-4bb1-98db-26de3448541b
agent_id: 8321f735-a52c-4f49-aca9-66f9266c5fe5
labels: "one=foo,two=bar"
agent_name: "My Agent"
tls_config:
  insecure: false
`

				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				err := os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				expectedConfig := &Config{
					Endpoint:  "localhost:1234",
					SecretKey: &secretKeyContents,
					AgentID:   "8321f735-a52c-4f49-aca9-66f9266c5fe5",
					Labels:    &labelsContents,
					AgentName: &agentNameContents,
					TLS: &TLSConfig{
						Insecure: false,
					},
				}

				cfg, err := ParseConfig(configPath)
				assert.NoError(t, err)
				assert.Equal(t, expectedConfig, cfg)
			},
		},
		{
			desc: "TLS Invalid CA File",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				configContents := `
endpoint: localhost:1234
secret_key: b92222ee-a1fc-4bb1-98db-26de3448541b
agent_id: 8321f735-a52c-4f49-aca9-66f9266c5fe5
labels: "one=foo,two=bar"
agent_name: "My Agent"
tls_config:
  insecure: false
  ca_file: /some/bad/file-ca.crt
`

				err := os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				cfg, err := ParseConfig(configPath)
				assert.ErrorContains(t, err, errInvalidCAFile)
				assert.Nil(t, cfg)
			},
		},
		{
			desc: "TLS Valid CA File",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				caPath := filepath.Join(tmpDir, "file-ca.crt")
				_, err := os.Create(caPath)
				require.NoError(t, err)

				configContents := fmt.Sprintf(`
endpoint: localhost:1234
secret_key: b92222ee-a1fc-4bb1-98db-26de3448541b
agent_id: 8321f735-a52c-4f49-aca9-66f9266c5fe5
labels: "one=foo,two=bar"
agent_name: "My Agent"
tls_config:
  insecure: false
  ca_file: %s
`, caPath)

				err = os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				expectedConfig := &Config{
					Endpoint:  "localhost:1234",
					SecretKey: &secretKeyContents,
					AgentID:   "8321f735-a52c-4f49-aca9-66f9266c5fe5",
					Labels:    &labelsContents,
					AgentName: &agentNameContents,
					TLS: &TLSConfig{
						Insecure: false,
						CAFile:   &caPath,
					},
				}

				cfg, err := ParseConfig(configPath)
				assert.NoError(t, err)
				assert.Equal(t, expectedConfig, cfg)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestCmpUpdatableFields(t *testing.T) {
	secretKeyContents := "b92222ee-a1fc-4bb1-98db-26de3448541b"
	nameOne, nameTwo := "one", "two"
	labelsOne, labelsTwo := "one=1", "two=2"
	testCase := []struct {
		desc    string
		baseCfg Config
		compare Config
		expect  bool
	}{
		{
			desc: "Full match",
			baseCfg: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   "20ce90b8-506c-4a3b-8134-21aa8d526e03",
				Labels:    &labelsOne,
				AgentName: &nameOne,
			},
			compare: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   "20ce90b8-506c-4a3b-8134-21aa8d526e03",
				Labels:    &labelsOne,
				AgentName: &nameOne,
			},
			expect: true,
		},
		{
			desc: "Only Updatable fields match",
			baseCfg: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   "20ce90b8-506c-4a3b-8134-21aa8d526e03",
				Labels:    &labelsOne,
				AgentName: &nameOne,
			},
			compare: Config{
				Endpoint:  "ws://some.host.com",
				SecretKey: nil,
				AgentID:   "d71cb88c-a4d3-4992-8bc8-d82702fdcb21",
				Labels:    &labelsOne,
				AgentName: &nameOne,
			},
			expect: true,
		},
		{
			desc: "Labels match no Agent Name",
			baseCfg: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   "20ce90b8-506c-4a3b-8134-21aa8d526e03",
				Labels:    &labelsOne,
				AgentName: nil,
			},
			compare: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   "20ce90b8-506c-4a3b-8134-21aa8d526e03",
				Labels:    &labelsOne,
				AgentName: nil,
			},
			expect: true,
		},
		{
			desc: "Labels don't match no Agent Name",
			baseCfg: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   "20ce90b8-506c-4a3b-8134-21aa8d526e03",
				Labels:    &labelsOne,
				AgentName: nil,
			},
			compare: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   "20ce90b8-506c-4a3b-8134-21aa8d526e03",
				Labels:    &labelsTwo,
				AgentName: nil,
			},
			expect: false,
		},
		{
			desc: "Agent Name match no labels",
			baseCfg: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   "20ce90b8-506c-4a3b-8134-21aa8d526e03",
				Labels:    nil,
				AgentName: &nameOne,
			},
			compare: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   "20ce90b8-506c-4a3b-8134-21aa8d526e03",
				Labels:    nil,
				AgentName: &nameOne,
			},
			expect: true,
		},
		{
			desc: "Agent Name doesn't match no labels",
			baseCfg: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   "20ce90b8-506c-4a3b-8134-21aa8d526e03",
				Labels:    nil,
				AgentName: &nameOne,
			},
			compare: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   "20ce90b8-506c-4a3b-8134-21aa8d526e03",
				Labels:    nil,
				AgentName: &nameTwo,
			},
			expect: false,
		},
		{
			desc: "Label present in base not in other",
			baseCfg: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   "20ce90b8-506c-4a3b-8134-21aa8d526e03",
				Labels:    &labelsOne,
				AgentName: nil,
			},
			compare: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   "20ce90b8-506c-4a3b-8134-21aa8d526e03",
				Labels:    nil,
				AgentName: nil,
			},
			expect: false,
		},
		{
			desc: "Label present in other not in base",
			baseCfg: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   "20ce90b8-506c-4a3b-8134-21aa8d526e03",
				Labels:    nil,
				AgentName: nil,
			},
			compare: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   "20ce90b8-506c-4a3b-8134-21aa8d526e03",
				Labels:    &labelsTwo,
				AgentName: nil,
			},
			expect: false,
		},
	}

	for _, tc := range testCase {
		t.Run(tc.desc, func(t *testing.T) {
			actual := tc.baseCfg.CmpUpdatableFields(tc.compare)
			assert.Equal(t, tc.expect, actual)
		})
	}
}

func TestGetSecretKey(t *testing.T) {
	secretKeyContents := "b92222ee-a1fc-4bb1-98db-26de3448541b"
	testCases := []struct {
		desc     string
		config   Config
		expected string
	}{
		{
			desc:     "Missing secretKey",
			config:   Config{},
			expected: "",
		},
		{
			desc: "Has secretKey",
			config: Config{
				SecretKey: &secretKeyContents,
			},
			expected: secretKeyContents,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			actual := tc.config.GetSecretKey()
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestConfigCopy(t *testing.T) {
	secretKeyContents := "b92222ee-a1fc-4bb1-98db-26de3448541b"
	labelsContents := "one=foo,two=bar"
	agentNameContents := "My Agent"
	keyFileContents := "My Key File"
	certFileContents := "My Cert File"
	caFileContents := "My CA File"

	tlscfg := TLSConfig{
		Insecure: false,
		KeyFile:  &keyFileContents,
		CertFile: &certFileContents,
		CAFile:   &caFileContents,
	}
	cfg := Config{
		Endpoint:  "ws://localhost:1234",
		SecretKey: &secretKeyContents,
		AgentID:   "20ce90b8-506c-4a3b-8134-21aa8d526e03",
		Labels:    &labelsContents,
		AgentName: &agentNameContents,
		TLS:       &tlscfg,
	}

	copyCfg := cfg.Copy()
	require.Equal(t, cfg, *copyCfg)
}
