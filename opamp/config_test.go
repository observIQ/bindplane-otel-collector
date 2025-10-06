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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Must is a helper function for tests that panics if there is an error creating the object of type T
func Must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

var testAgentIDString = "01HX2DWEQZ045KQR3VG0EYEZ94"
var testAgentID = Must(ParseAgentID(testAgentIDString))

func TestAgentIDFromInstanceUid(t *testing.T) {
	t.Run("UUID instance_uid", func(t *testing.T) {
		// Create a UUID
		u := uuid.New()
		instanceUID := [16]byte(u)

		// Convert to AgentID
		agentID, err := AgentIDFromInstanceUID(instanceUID)
		require.NoError(t, err)

		// Verify it's a UUID type
		assert.Equal(t, "UUID", agentID.Type())
		assert.Equal(t, u.String(), agentID.String())

		// Verify round-trip conversion
		assert.Equal(t, instanceUID, [16]byte(agentID.OpAMPInstanceUID()))
	})

	t.Run("ULID instance_uid", func(t *testing.T) {
		// Use the test ULID - note that ULID can be parsed as UUID v7
		// so we just verify the conversion works correctly
		instanceUID := testAgentID.OpAMPInstanceUID()

		// Convert to AgentID
		agentID, err := AgentIDFromInstanceUID([16]byte(instanceUID))
		require.NoError(t, err)

		// The type might be UUID (v7) since UUID parsing succeeds for ULID bytes
		// What matters is that we can round-trip the value
		assert.Contains(t, []string{"UUID", "ULID"}, agentID.Type())

		// Verify round-trip conversion works
		assert.Equal(t, [16]byte(instanceUID), [16]byte(agentID.OpAMPInstanceUID()))
	})
}

func TestToTLS(t *testing.T) {
	invalidCAFile := "/some/bad/file-ca.crt"
	invalidKeyFile := "/some/bad/file.key"
	invalidCertFile := "/some/bad/file.crt"
	caFileContents := "./testdata/test-ca.crt"
	keyFileContents := "./testdata/test.key"
	certFileContents := "./testdata/test.crt"

	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "No TLS Provided",
			testFunc: func(t *testing.T) {
				cfg := Config{
					TLS: nil,
				}

				actual, err := cfg.ToTLS(nil)
				assert.NoError(t, err)
				assert.Nil(t, actual)
			},
		},
		{
			desc: "TLS Insecure",
			testFunc: func(t *testing.T) {
				cfg := Config{
					TLS: &TLSConfig{
						InsecureSkipVerify: true,
					},
				}

				expectedConfig := tls.Config{
					InsecureSkipVerify: true,
					MinVersion:         tls.VersionTLS12,
				}

				actual, err := cfg.ToTLS(nil)
				assert.NoError(t, err)
				assert.Equal(t, &expectedConfig, actual)
			},
		},
		{
			desc: "TLS Insecure with mtls",
			testFunc: func(t *testing.T) {
				cfg := Config{
					TLS: &TLSConfig{
						InsecureSkipVerify: true,
						KeyFile:            &keyFileContents,
						CertFile:           &certFileContents,
					},
				}

				expectedConfig := tls.Config{
					InsecureSkipVerify: true,
					MinVersion:         tls.VersionTLS12,
				}

				cert, err := tls.LoadX509KeyPair(certFileContents, keyFileContents)
				require.NoError(t, err)
				expectedConfig.Certificates = []tls.Certificate{cert}

				actual, err := cfg.ToTLS(nil)
				assert.NoError(t, err)
				assert.Equal(t, &expectedConfig, actual)
			},
		},
		{
			desc: "Insecure False, No Files Specified",
			testFunc: func(t *testing.T) {
				cfg := Config{
					TLS: &TLSConfig{
						InsecureSkipVerify: false,
					},
				}

				expectedConfig := tls.Config{
					MinVersion: tls.VersionTLS12,
				}

				actual, err := cfg.ToTLS(nil)
				assert.NoError(t, err)
				assert.Equal(t, &expectedConfig, actual)
			},
		},
		{
			desc: "Insecure False, Invalid CA File Specified",
			testFunc: func(t *testing.T) {
				cfg := Config{
					TLS: &TLSConfig{
						InsecureSkipVerify: false,
						CAFile:             &invalidCAFile,
					},
				}

				actual, err := cfg.ToTLS(nil)
				assert.ErrorContains(t, err, "failed to read CA file")
				assert.Nil(t, actual)
			},
		},
		{
			desc: "Insecure False, Valid CA File Specified",
			testFunc: func(t *testing.T) {
				cfg := Config{
					TLS: &TLSConfig{
						InsecureSkipVerify: false,
						CAFile:             &caFileContents,
					},
				}

				actual, err := cfg.ToTLS(nil)
				assert.NoError(t, err)
				assert.NotNil(t, actual)
				assert.False(t, actual.InsecureSkipVerify)

				// Can't compare root CA's due to embedded function in Cert Pool structure
			},
		},
		{
			desc: "Insecure False, Invalid Key and Cert Files Specified",
			testFunc: func(t *testing.T) {
				cfg := Config{
					TLS: &TLSConfig{
						InsecureSkipVerify: false,
						KeyFile:            &invalidKeyFile,
						CertFile:           &invalidCertFile,
					},
				}

				_, err := tls.LoadX509KeyPair(invalidCertFile, invalidKeyFile)
				errinvalidKeyorCertFile := fmt.Sprintf("failed to read Key and Cert file: %s", err)

				actual, err := cfg.ToTLS(nil)
				assert.ErrorContains(t, err, errinvalidKeyorCertFile)
				assert.Nil(t, actual)
			},
		},
		{
			desc: "Insecure False, Valid Key and Cert Files Specified",
			testFunc: func(t *testing.T) {
				cfg := Config{
					TLS: &TLSConfig{
						InsecureSkipVerify: false,
						KeyFile:            &keyFileContents,
						CertFile:           &certFileContents,
					},
				}

				expectedConfig := tls.Config{
					MinVersion: tls.VersionTLS12,
				}

				cert, err := tls.LoadX509KeyPair(certFileContents, keyFileContents)
				require.NoError(t, err)
				expectedConfig.Certificates = []tls.Certificate{cert}

				actual, err := cfg.ToTLS(nil)
				assert.NoError(t, err)
				assert.Equal(t, &expectedConfig, actual)
			},
		},
		{
			desc: "Insecure False, All TLS Files Valid and Specified",
			testFunc: func(t *testing.T) {
				cfg := Config{
					TLS: &TLSConfig{
						InsecureSkipVerify: false,
						CAFile:             &caFileContents,
						KeyFile:            &keyFileContents,
						CertFile:           &certFileContents,
					},
				}

				expectedConfig := tls.Config{
					MinVersion: tls.VersionTLS12,
				}

				caCert, err := os.ReadFile(caFileContents)
				require.NoError(t, err)
				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM(caCert)
				expectedConfig.RootCAs = caCertPool

				cert, err := tls.LoadX509KeyPair(certFileContents, keyFileContents)
				require.NoError(t, err)
				expectedConfig.Certificates = []tls.Certificate{cert}

				actual, err := cfg.ToTLS(caCertPool)
				assert.NoError(t, err)

				assert.Equal(t, &expectedConfig, actual)
			},
		},
		{
			desc: "Insecure True, All TLS Files Valid and Specified",
			testFunc: func(t *testing.T) {
				cfg := Config{
					TLS: &TLSConfig{
						InsecureSkipVerify: true,
						CAFile:             &caFileContents,
						KeyFile:            &keyFileContents,
						CertFile:           &certFileContents,
					},
				}

				expectedConfig := tls.Config{
					InsecureSkipVerify: true,
					MinVersion:         tls.VersionTLS12,
				}

				caCert, err := os.ReadFile(caFileContents)
				require.NoError(t, err)
				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM(caCert)
				expectedConfig.RootCAs = caCertPool

				cert, err := tls.LoadX509KeyPair(certFileContents, keyFileContents)
				require.NoError(t, err)
				expectedConfig.Certificates = []tls.Certificate{cert}

				actual, err := cfg.ToTLS(caCertPool)
				assert.NoError(t, err)

				assert.Equal(t, &expectedConfig, actual)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

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
				assert.ErrorContains(t, err, errPrefixReadFile)
				assert.Nil(t, cfg)
			},
		},
		{
			desc: "Successful Full Parse",
			testFunc: func(t *testing.T) {
				configContents := fmt.Sprintf(`
endpoint: localhost:1234
secret_key: b92222ee-a1fc-4bb1-98db-26de3448541b
agent_id: %s
labels: "one=foo,two=bar"
agent_name: "My Agent"
`, testAgentIDString)

				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				err := os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				expectedConfig := &Config{
					Endpoint:  "localhost:1234",
					SecretKey: &secretKeyContents,
					AgentID:   testAgentID,
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
				configContents := fmt.Sprintf(`
endpoint: localhost:1234
agent_id: %s
`, testAgentIDString)

				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				err := os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				expectedConfig := &Config{
					Endpoint:  "localhost:1234",
					SecretKey: nil,
					AgentID:   testAgentID,
					Labels:    nil,
					AgentName: nil,
				}

				cfg, err := ParseConfig(configPath)
				assert.NoError(t, err)
				assert.Equal(t, expectedConfig, cfg)
			},
		},
		{
			desc: "Successful Full Parse with TLS Insecure Skip Verify",
			testFunc: func(t *testing.T) {
				configContents := fmt.Sprintf(`
endpoint: localhost:1234
secret_key: b92222ee-a1fc-4bb1-98db-26de3448541b
agent_id: %s
labels: "one=foo,two=bar"
agent_name: "My Agent"
tls_config:
  insecure_skip_verify: true
`, testAgentIDString)

				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				err := os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				expectedConfig := &Config{
					Endpoint:  "localhost:1234",
					SecretKey: &secretKeyContents,
					AgentID:   testAgentID,
					Labels:    &labelsContents,
					AgentName: &agentNameContents,
					TLS: &TLSConfig{
						InsecureSkipVerify: true,
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
				configContents := fmt.Sprintf(`
endpoint: localhost:1234
secret_key: b92222ee-a1fc-4bb1-98db-26de3448541b
agent_id: %s
labels: "one=foo,two=bar"
agent_name: "My Agent"
tls_config:
  insecure_skip_verify: false
`, testAgentIDString)

				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				err := os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				expectedConfig := &Config{
					Endpoint:  "localhost:1234",
					SecretKey: &secretKeyContents,
					AgentID:   testAgentID,
					Labels:    &labelsContents,
					AgentName: &agentNameContents,
					TLS: &TLSConfig{
						InsecureSkipVerify: false,
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
agent_id: 01HX2DWEQZ045KQR3VG0EYEZ94
labels: "one=foo,two=bar"
agent_name: "My Agent"
tls_config:
  insecure_skip_verify: false
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
				f, err := os.Create(caPath)
				require.NoError(t, err)
				defer f.Close()

				configContents := fmt.Sprintf(`
endpoint: localhost:1234
secret_key: b92222ee-a1fc-4bb1-98db-26de3448541b
agent_id: %s
labels: "one=foo,two=bar"
agent_name: "My Agent"
tls_config:
  insecure_skip_verify: false
  ca_file: %s
`, testAgentIDString, caPath)

				err = os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				expectedConfig := &Config{
					Endpoint:  "localhost:1234",
					SecretKey: &secretKeyContents,
					AgentID:   testAgentID,
					Labels:    &labelsContents,
					AgentName: &agentNameContents,
					TLS: &TLSConfig{
						InsecureSkipVerify: false,
						CAFile:             &caPath,
					},
				}

				cfg, err := ParseConfig(configPath)
				assert.NoError(t, err)
				assert.Equal(t, expectedConfig, cfg)
			},
		},
		{
			desc: "TLS Invalid Key and Cert File",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				configContents := `
endpoint: localhost:1234
secret_key: b92222ee-a1fc-4bb1-98db-26de3448541b
agent_id: 01HX2DWEQZ045KQR3VG0EYEZ94
labels: "one=foo,two=bar"
agent_name: "My Agent"
tls_config:
  insecure_skip_verify: false
  key_file: /some/bad/file.key
  cert_file: /some/bad/file.crt
`

				err := os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				cfg, err := ParseConfig(configPath)
				assert.ErrorContains(t, err, errInvalidKeyFile)
				assert.Nil(t, cfg)
			},
		},
		{
			desc: "TLS Valid Key File Invalid Cert File",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				keyPath := filepath.Join(tmpDir, "file-key.crt")
				k, err := os.Create(keyPath)
				require.NoError(t, err)
				defer k.Close()

				configContents := fmt.Sprintf(`
endpoint: localhost:1234
secret_key: b92222ee-a1fc-4bb1-98db-26de3448541b
agent_id: 01HX2DWEQZ045KQR3VG0EYEZ94
labels: "one=foo,two=bar"
agent_name: "My Agent"
tls_config:
  insecure_skip_verify: false
  key_file: %s
  cert_file: /some/bad/file.crt
`, keyPath)

				err = os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				cfg, err := ParseConfig(configPath)
				assert.ErrorContains(t, err, errInvalidCertFile)
				assert.Nil(t, cfg)
			},
		},
		{
			desc: "TLS Valid Cert File Invalid Key File",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				certPath := filepath.Join(tmpDir, "file-cert.crt")
				c, err := os.Create(certPath)
				require.NoError(t, err)
				defer c.Close()

				configContents := fmt.Sprintf(`
endpoint: localhost:1234
secret_key: b92222ee-a1fc-4bb1-98db-26de3448541b
agent_id: 01HX2DWEQZ045KQR3VG0EYEZ94
labels: "one=foo,two=bar"
agent_name: "My Agent"
tls_config:
  insecure_skip_verify: false
  key_file: /some/bad/file.key
  cert_file: %s
`, certPath)

				err = os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				cfg, err := ParseConfig(configPath)
				assert.ErrorContains(t, err, errInvalidKeyFile)
				assert.Nil(t, cfg)
			},
		},
		{
			desc: "TLS Only Valid Key File Provided",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				keyPath := filepath.Join(tmpDir, "file-cert.crt")
				k, err := os.Create(keyPath)
				require.NoError(t, err)
				defer k.Close()

				configContents := fmt.Sprintf(`
endpoint: localhost:1234
secret_key: b92222ee-a1fc-4bb1-98db-26de3448541b
agent_id: 01HX2DWEQZ045KQR3VG0EYEZ94
labels: "one=foo,two=bar"
agent_name: "My Agent"
tls_config:
  insecure_skip_verify: false
  key_file: %s
`, keyPath)

				err = os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				cfg, err := ParseConfig(configPath)
				assert.ErrorContains(t, err, errMissingTLSFiles)
				assert.Nil(t, cfg)
			},
		},
		{
			desc: "TLS Only Valid Cert File Provided",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				certPath := filepath.Join(tmpDir, "file-cert.crt")
				c, err := os.Create(certPath)
				require.NoError(t, err)
				defer c.Close()

				configContents := fmt.Sprintf(`
endpoint: localhost:1234
secret_key: b92222ee-a1fc-4bb1-98db-26de3448541b
agent_id: 01HX2DWEQZ045KQR3VG0EYEZ94
labels: "one=foo,two=bar"
agent_name: "My Agent"
tls_config:
  insecure_skip_verify: false
  cert_file: %s
`, certPath)

				err = os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				cfg, err := ParseConfig(configPath)
				assert.ErrorContains(t, err, errMissingTLSFiles)
				assert.Nil(t, cfg)
			},
		},
		{
			desc: "Successful Full Parse with TLS Valid Key and Cert Files",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				keyPath := filepath.Join(tmpDir, "file-key.crt")
				k, err := os.Create(keyPath)
				require.NoError(t, err)
				defer k.Close()

				certPath := filepath.Join(tmpDir, "file-cert.crt")
				c, err := os.Create(certPath)
				require.NoError(t, err)
				defer c.Close()

				configContents := fmt.Sprintf(`
endpoint: localhost:1234
secret_key: b92222ee-a1fc-4bb1-98db-26de3448541b
agent_id: %s
labels: "one=foo,two=bar"
agent_name: "My Agent"
tls_config:
  insecure_skip_verify: false
  key_file: %s
  cert_file: %s
`, testAgentIDString, keyPath, certPath)

				err = os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				expectedConfig := &Config{
					Endpoint:  "localhost:1234",
					SecretKey: &secretKeyContents,
					AgentID:   testAgentID,
					Labels:    &labelsContents,
					AgentName: &agentNameContents,
					TLS: &TLSConfig{
						InsecureSkipVerify: false,
						KeyFile:            &keyPath,
						CertFile:           &certPath,
					},
				}

				cfg, err := ParseConfig(configPath)
				assert.NoError(t, err)
				assert.Equal(t, expectedConfig, cfg)
			},
		},
		{
			desc: "Successful Parse With Environment Variables",
			testFunc: func(t *testing.T) {
				endpointEnvVar := "TEST_ENDPOINT"
				require.NoError(t, os.Setenv(endpointEnvVar, "localhost:1234"))
				defer func() {
					require.NoError(t, os.Unsetenv(endpointEnvVar))
				}()

				secretKeyEnvVar := "TEST_SECRET_KEY"
				require.NoError(t, os.Setenv(secretKeyEnvVar, secretKeyContents))
				defer func() {
					require.NoError(t, os.Unsetenv(secretKeyEnvVar))
				}()

				agentIDEnvVar := "TEST_AGENT_ID"
				require.NoError(t, os.Setenv(agentIDEnvVar, testAgentIDString))
				defer func() {
					require.NoError(t, os.Unsetenv(agentIDEnvVar))
				}()

				labelsEnvVar := "TEST_LABELS"
				require.NoError(t, os.Setenv(labelsEnvVar, "one=foo,two=bar"))
				defer func() {
					require.NoError(t, os.Unsetenv(labelsEnvVar))
				}()

				agentNameEnvVar := "TEST_AGENT_NAME"
				require.NoError(t, os.Setenv(agentNameEnvVar, "My Agent"))
				defer func() {
					require.NoError(t, os.Unsetenv(agentNameEnvVar))
				}()

				configContents := fmt.Sprintf(`
endpoint: ${%s}
secret_key: ${%s}
agent_id: ${%s}
labels: ${%s}
agent_name: ${%s}
`, endpointEnvVar, secretKeyEnvVar, agentIDEnvVar, labelsEnvVar, agentNameEnvVar)

				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				err := os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				expectedConfig := &Config{
					Endpoint:  "localhost:1234",
					SecretKey: &secretKeyContents,
					AgentID:   testAgentID,
					Labels:    &labelsContents,
					AgentName: &agentNameContents,
				}

				cfg, err := ParseConfig(configPath)
				assert.NoError(t, err)
				assert.Equal(t, expectedConfig, cfg)
			},
		},
		{
			desc: "Successful Parse With env:Environment Variables",
			testFunc: func(t *testing.T) {
				endpointEnvVar := "TEST_ENDPOINT"
				require.NoError(t, os.Setenv(endpointEnvVar, "localhost:1234"))
				defer func() {
					require.NoError(t, os.Unsetenv(endpointEnvVar))
				}()

				secretKeyEnvVar := "TEST_SECRET_KEY"
				require.NoError(t, os.Setenv(secretKeyEnvVar, secretKeyContents))
				defer func() {
					require.NoError(t, os.Unsetenv(secretKeyEnvVar))
				}()

				agentIDEnvVar := "TEST_AGENT_ID"
				require.NoError(t, os.Setenv(agentIDEnvVar, testAgentIDString))
				defer func() {
					require.NoError(t, os.Unsetenv(agentIDEnvVar))
				}()

				labelsEnvVar := "TEST_LABELS"
				require.NoError(t, os.Setenv(labelsEnvVar, "one=foo,two=bar"))
				defer func() {
					require.NoError(t, os.Unsetenv(labelsEnvVar))
				}()

				agentNameEnvVar := "TEST_AGENT_NAME"
				require.NoError(t, os.Setenv(agentNameEnvVar, "My Agent"))
				defer func() {
					require.NoError(t, os.Unsetenv(agentNameEnvVar))
				}()

				configContents := fmt.Sprintf(`
endpoint: ${env:%s}
secret_key: ${env:%s}
agent_id: ${env:%s}
labels: ${env:%s}
agent_name: ${env:%s}
`, endpointEnvVar, secretKeyEnvVar, agentIDEnvVar, labelsEnvVar, agentNameEnvVar)

				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				err := os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				expectedConfig := &Config{
					Endpoint:  "localhost:1234",
					SecretKey: &secretKeyContents,
					AgentID:   testAgentID,
					Labels:    &labelsContents,
					AgentName: &agentNameContents,
				}

				cfg, err := ParseConfig(configPath)
				assert.NoError(t, err)
				assert.Equal(t, expectedConfig, cfg)
			},
		},
		{
			desc: "Successful Full Parse with TLS Valid Key and Cert Environment Variables",
			testFunc: func(t *testing.T) {
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "manager.yml")

				keyPath := filepath.Join(tmpDir, "file-key.crt")
				k, err := os.Create(keyPath)
				require.NoError(t, err)
				defer k.Close()

				certPath := filepath.Join(tmpDir, "file-cert.crt")
				c, err := os.Create(certPath)
				require.NoError(t, err)
				defer c.Close()

				keyEnvVariable := "TEST_TLS_KEY"
				require.NoError(t, os.Setenv(keyEnvVariable, keyPath))
				defer func() {
					require.NoError(t, os.Unsetenv(keyEnvVariable))
				}()

				certEnvVariable := "TEST_TLS_CERT"
				require.NoError(t, os.Setenv(certEnvVariable, certPath))
				defer func() {
					require.NoError(t, os.Unsetenv(certEnvVariable))
				}()

				configContents := fmt.Sprintf(`
endpoint: localhost:1234
secret_key: b92222ee-a1fc-4bb1-98db-26de3448541b
agent_id: %s
labels: "one=foo,two=bar"
agent_name: "My Agent"
tls_config:
  insecure_skip_verify: false
  key_file: ${%s}
  cert_file: ${%s}
`, testAgentIDString, keyEnvVariable, certEnvVariable)

				err = os.WriteFile(configPath, []byte(configContents), os.ModePerm)
				require.NoError(t, err)

				expectedConfig := &Config{
					Endpoint:  "localhost:1234",
					SecretKey: &secretKeyContents,
					AgentID:   testAgentID,
					Labels:    &labelsContents,
					AgentName: &agentNameContents,
					TLS: &TLSConfig{
						InsecureSkipVerify: false,
						KeyFile:            &keyPath,
						CertFile:           &certPath,
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
				AgentID:   testAgentID,
				Labels:    &labelsOne,
				AgentName: &nameOne,
			},
			compare: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   testAgentID,
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
				AgentID:   testAgentID,
				Labels:    &labelsOne,
				AgentName: &nameOne,
			},
			compare: Config{
				Endpoint:  "ws://some.host.com",
				SecretKey: nil,
				AgentID:   EmptyAgentID,
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
				AgentID:   testAgentID,
				Labels:    &labelsOne,
				AgentName: nil,
			},
			compare: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   testAgentID,
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
				AgentID:   testAgentID,
				Labels:    &labelsOne,
				AgentName: nil,
			},
			compare: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   testAgentID,
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
				AgentID:   testAgentID,
				Labels:    nil,
				AgentName: &nameOne,
			},
			compare: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   testAgentID,
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
				AgentID:   testAgentID,
				Labels:    nil,
				AgentName: &nameOne,
			},
			compare: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   testAgentID,
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
				AgentID:   testAgentID,
				Labels:    &labelsOne,
				AgentName: nil,
			},
			compare: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   testAgentID,
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
				AgentID:   testAgentID,
				Labels:    nil,
				AgentName: nil,
			},
			compare: Config{
				Endpoint:  "ws://localhost:1234",
				SecretKey: &secretKeyContents,
				AgentID:   testAgentID,
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
		InsecureSkipVerify: false,
		KeyFile:            &keyFileContents,
		CertFile:           &certFileContents,
		CAFile:             &caFileContents,
	}
	cfg := Config{
		Endpoint:  "ws://localhost:1234",
		SecretKey: &secretKeyContents,
		AgentID:   testAgentID,
		Labels:    &labelsContents,
		AgentName: &agentNameContents,
		TLS:       &tlscfg,
	}

	copyCfg := cfg.Copy()
	require.Equal(t, cfg, *copyCfg)
}

func TestParseAgentID(t *testing.T) {
	testCases := []struct {
		name        string
		id          string
		expected    AgentID
		expectedErr string
	}{
		{
			name: "Valid ULID",
			id:   "01J9RQ8V3ZT95MRH05DKJA3KSM",
			expected: AgentID{
				by:     [16]byte{0x1, 0x92, 0x71, 0x74, 0x6c, 0x7f, 0xd2, 0x4b, 0x4c, 0x44, 0x5, 0x6c, 0xe4, 0xa1, 0xcf, 0x34},
				idType: agentIDTypeULID,
				orig:   "01J9RQ8V3ZT95MRH05DKJA3KSM",
			},
		},
		{
			name: "Valid UUID",
			id:   "01927175-7a98-7585-94ce-cc833ee7735d",
			expected: AgentID{
				by:     [16]byte{0x1, 0x92, 0x71, 0x75, 0x7a, 0x98, 0x75, 0x85, 0x94, 0xce, 0xcc, 0x83, 0x3e, 0xe7, 0x73, 0x5d},
				idType: agentIDTypeUUID,
				orig:   "01927175-7a98-7585-94ce-cc833ee7735d",
			},
		},
		{
			name:        "Invalid ULID",
			id:          "A1J9RQ8V3ZT95MRH05DKJA3KSM",
			expectedErr: "parse ulid:",
		},
		{
			name:        "Invalid UUID",
			id:          "01927175-7a98-7585-94ce-cc833ee7735l",
			expectedErr: "parse uuid:",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := ParseAgentID(tc.id)
			if tc.expectedErr != "" {
				require.ErrorContains(t, err, tc.expectedErr)
				return
			}

			require.Equal(t, tc.expected, id)
		})
	}
}

func TestAgentIDFromUUID(t *testing.T) {
	id := AgentIDFromUUID(Must(uuid.Parse("01927175-7a98-7585-94ce-cc833ee7735d")))
	require.Equal(t, AgentID{
		by:     [16]byte{0x1, 0x92, 0x71, 0x75, 0x7a, 0x98, 0x75, 0x85, 0x94, 0xce, 0xcc, 0x83, 0x3e, 0xe7, 0x73, 0x5d},
		idType: agentIDTypeUUID,
		orig:   "01927175-7a98-7585-94ce-cc833ee7735d",
	}, id)
}

func TestAgentID_String(t *testing.T) {
	uuidID := Must(ParseAgentID("01927175-7a98-7585-94ce-cc833ee7735d"))
	ulidID := Must(ParseAgentID("01J9RQ8V3ZT95MRH05DKJA3KSM"))

	require.Equal(t, "01927175-7a98-7585-94ce-cc833ee7735d", uuidID.String())
	require.Equal(t, "01J9RQ8V3ZT95MRH05DKJA3KSM", ulidID.String())
}

func TestAgentID_OpAMPInstanceUID(t *testing.T) {
	uuidID := Must(ParseAgentID("01927175-7a98-7585-94ce-cc833ee7735d"))
	ulidID := Must(ParseAgentID("01J9RQ8V3ZT95MRH05DKJA3KSM"))

	require.EqualValues(t,
		[16]byte{0x1, 0x92, 0x71, 0x75, 0x7a, 0x98, 0x75, 0x85, 0x94, 0xce, 0xcc, 0x83, 0x3e, 0xe7, 0x73, 0x5d},
		uuidID.OpAMPInstanceUID(),
	)

	require.EqualValues(t,
		[16]byte{0x1, 0x92, 0x71, 0x74, 0x6c, 0x7f, 0xd2, 0x4b, 0x4c, 0x44, 0x5, 0x6c, 0xe4, 0xa1, 0xcf, 0x34},
		ulidID.OpAMPInstanceUID(),
	)
}

func TestAgentID_Type(t *testing.T) {
	uuidID := Must(ParseAgentID("01927175-7a98-7585-94ce-cc833ee7735d"))
	ulidID := Must(ParseAgentID("01J9RQ8V3ZT95MRH05DKJA3KSM"))

	require.EqualValues(t, agentIDTypeUUID, uuidID.Type())
	require.EqualValues(t, agentIDTypeULID, ulidID.Type())
}

func TestAgentID_MarshalYaml(t *testing.T) {
	uuidID := Must(ParseAgentID("01927175-7a98-7585-94ce-cc833ee7735d"))
	ulidID := Must(ParseAgentID("01J9RQ8V3ZT95MRH05DKJA3KSM"))

	uuidYaml, err := yaml.Marshal(uuidID)
	require.NoError(t, err)
	require.Equal(t, "01927175-7a98-7585-94ce-cc833ee7735d\n", string(uuidYaml))

	ulidYaml, err := yaml.Marshal(ulidID)
	require.NoError(t, err)
	require.Equal(t, "01J9RQ8V3ZT95MRH05DKJA3KSM\n", string(ulidYaml))
}

func TestAgentID_UnmarshalYaml(t *testing.T) {
	t.Run("UUID", func(t *testing.T) {
		var uuidAgentID AgentID
		err := yaml.Unmarshal([]byte("01927175-7a98-7585-94ce-cc833ee7735d"), &uuidAgentID)
		require.NoError(t, err)
		require.Equal(t, AgentID{
			by:     [16]byte{0x1, 0x92, 0x71, 0x75, 0x7a, 0x98, 0x75, 0x85, 0x94, 0xce, 0xcc, 0x83, 0x3e, 0xe7, 0x73, 0x5d},
			idType: agentIDTypeUUID,
			orig:   "01927175-7a98-7585-94ce-cc833ee7735d",
		}, uuidAgentID)
	})

	t.Run("ULID", func(t *testing.T) {
		var ulidAgentID AgentID
		err := yaml.Unmarshal([]byte("01J9RQ8V3ZT95MRH05DKJA3KSM"), &ulidAgentID)
		require.NoError(t, err)
		require.Equal(t, AgentID{
			by:     [16]byte{0x1, 0x92, 0x71, 0x74, 0x6c, 0x7f, 0xd2, 0x4b, 0x4c, 0x44, 0x5, 0x6c, 0xe4, 0xa1, 0xcf, 0x34},
			idType: agentIDTypeULID,
			orig:   "01J9RQ8V3ZT95MRH05DKJA3KSM",
		}, ulidAgentID)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		// Invalid IDs will give an empty ID instead of an error, so the
		// ID can be regenerated from the partially read config.
		var invalidID AgentID
		err := yaml.Unmarshal([]byte("Invalid"), &invalidID)
		require.NoError(t, err)
		require.Equal(t, EmptyAgentID, invalidID)
	})

	t.Run("Empty ID", func(t *testing.T) {
		var emptyID AgentID
		err := yaml.Unmarshal([]byte(`""`), &emptyID)
		require.NoError(t, err)
		require.Equal(t, EmptyAgentID, emptyID)
	})
}

func Test_cmpStringPtr(t *testing.T) {
	strPtr := func(s string) *string {
		return &s
	}
	type args struct {
		p1 *string
		p2 *string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{

		{
			name: "Both nil",
			args: args{
				p1: nil,
				p2: nil,
			},
			want: true,
		},
		{
			name: "p1 nil",
			args: args{
				p1: nil,
				p2: strPtr("foo"),
			},
			want: false,
		},
		{
			name: "p2 nil",
			args: args{
				p1: strPtr("foo"),
				p2: nil,
			},
			want: false,
		},
		{
			name: "Both empty",
			args: args{
				p1: strPtr(""),
				p2: strPtr(""),
			},
			want: true,
		},
		{
			name: "Both equal",
			args: args{
				p1: strPtr("foo"),
				p2: strPtr("foo"),
			},
			want: true,
		},
		{
			name: "Different",
			args: args{
				p1: strPtr("foo"),
				p2: strPtr("bar"),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CmpStringPtr(tt.args.p1, tt.args.p2); got != tt.want {
				t.Errorf("cmpStringPtr() = %v, want %v", got, tt.want)
			}
		})
	}
}
