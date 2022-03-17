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
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/helper"
	"github.com/open-telemetry/opentelemetry-log-collection/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPInputConfig(t *testing.T) {
	cfg := NewHTTPInputConfig("test_http")
	require.Equal(t, "test_http", cfg.ID())
	require.EqualValues(t, time.Second*60, cfg.IdleTimeout.Raw(), "expect idle timeout 60 seconds")
	require.EqualValues(t, time.Second*20, cfg.ReadTimeout.Raw(), "expect read timeout 20 second")
	require.EqualValues(t, time.Second*20, cfg.WriteTimeout.Raw(), "expect write timeout 20 seconds")
	require.Equal(t, helper.ByteSize(http.DefaultMaxHeaderBytes), cfg.MaxHeaderSize, "expect max header size 20 bytes")
	require.EqualValues(t, 10000000, cfg.MaxBodySize, "expected max body size 10mb")
}

func TestBuildOperator(t *testing.T) {
	cases := []struct {
		name      string
		input     func() (*HTTPInputConfig, func() error, error)
		expectErr bool
		errSubStr string
	}{
		{
			"default-with-auto-address",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = ":0"
				return cfg, nil, nil
			},
			false,
			"",
		},
		{
			"basic-auth",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = ":0"
				cfg.AuthConfig.Username = devUser
				cfg.AuthConfig.Password = "dev-password"
				return cfg, nil, nil
			},
			false,
			"",
		},
		{
			"basic-auth-missing-password",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = ":0"
				cfg.AuthConfig.Username = devUser
				return cfg, nil, nil
			},
			true,
			"password must be set when basic auth username is set",
		},
		{
			"basic-auth-missing-username",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = ":0"
				cfg.AuthConfig.Password = devUser
				return cfg, nil, nil
			},
			true,
			"username must be set when basic auth password is set",
		},
		{
			"multi-auth",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = ":0"
				cfg.AuthConfig.Username = "stanza"
				cfg.AuthConfig.Password = devUser
				cfg.AuthConfig.TokenHeader = authHeader
				cfg.AuthConfig.Tokens = []string{"token-a", "token-b"}
				return cfg, nil, nil
			},
			true,
			"token auth and basic auth cannot be enabled at the same time",
		},
		{
			"token-auth",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = ":0"
				cfg.AuthConfig.TokenHeader = authHeader
				cfg.AuthConfig.Tokens = []string{"token-a", "token-b"}
				return cfg, nil, nil
			},
			false,
			"",
		},
		{
			"token-auth-missing-tokens",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = ":0"
				cfg.AuthConfig.TokenHeader = authHeader
				return cfg, nil, nil
			},
			true,
			"auth.tokens is a required parameter when auth.token_header is set",
		},
		{
			"localhost-address",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				return cfg, nil, nil
			},
			false,
			"",
		},
		{
			"port-only",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = ":9000"
				return cfg, nil, nil
			},
			false,
			"",
		},
		{
			"address-port",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = "192.168.40.3:9090"
				return cfg, nil, nil
			},
			false,
			"",
		},
		{
			"no-address",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = ""
				return cfg, nil, nil
			},
			true,
			"missing required parameter 'listen_address'",
		},
		{
			"no-port",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = localHost
				return cfg, nil, nil
			},
			true,
			"failed to resolve listen_address: address localhost: missing port in address",
		},
		{
			"tls-default-version",
			func() (*HTTPInputConfig, func() error, error) {
				crt, key, cleanup, err := createTestCert()
				if err != nil {
					return nil, nil, err
				}

				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.TLS = TLSConfig{
					Enable:      true,
					Certificate: crt,
					PrivateKey:  key,
				}

				return cfg, cleanup, nil
			},
			false,
			"",
		},
		{
			"tls-1.0",
			func() (*HTTPInputConfig, func() error, error) {
				crt, key, cleanup, err := createTestCert()
				if err != nil {
					return nil, nil, err
				}

				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.TLS = TLSConfig{
					Enable:      true,
					Certificate: crt,
					PrivateKey:  key,
					MinVersion:  1.0,
				}

				return cfg, cleanup, nil
			},
			false,
			"",
		},
		{
			"tls-1.1",
			func() (*HTTPInputConfig, func() error, error) {
				crt, key, cleanup, err := createTestCert()
				if err != nil {
					return nil, nil, err
				}

				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.TLS = TLSConfig{
					Enable:      true,
					Certificate: crt,
					PrivateKey:  key,
					MinVersion:  1.1,
				}

				return cfg, cleanup, nil
			},
			false,
			"",
		},
		{
			"tls-1.2",
			func() (*HTTPInputConfig, func() error, error) {
				crt, key, cleanup, err := createTestCert()
				if err != nil {
					return nil, nil, err
				}

				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.TLS = TLSConfig{
					Enable:      true,
					Certificate: crt,
					PrivateKey:  key,
					MinVersion:  1.2,
				}

				return cfg, cleanup, nil
			},
			false,
			"",
		},
		{
			"tls-1.3",
			func() (*HTTPInputConfig, func() error, error) {
				crt, key, cleanup, err := createTestCert()
				if err != nil {
					return nil, nil, err
				}

				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.TLS = TLSConfig{
					Enable:      true,
					Certificate: crt,
					PrivateKey:  key,
					MinVersion:  1.3,
				}

				return cfg, cleanup, nil
			},
			false,
			"",
		},
		{
			"tls-disabled-with-config",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.TLS = TLSConfig{
					Enable:      false,
					Certificate: "/tmp/crt",
					PrivateKey:  "/tmp/key",
					MinVersion:  1.3,
				}

				return cfg, nil, nil
			},
			false,
			"",
		},
		{
			"invalid-tls-version",
			func() (*HTTPInputConfig, func() error, error) {
				crt, key, cleanup, err := createTestCert()
				if err != nil {
					return nil, nil, err
				}

				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.TLS = TLSConfig{
					Enable:      true,
					Certificate: crt,
					PrivateKey:  key,
					MinVersion:  1.4,
				}

				return cfg, cleanup, nil
			},
			true,
			"unsupported tls version",
		},
		{
			"missing-certificate-file",
			func() (*HTTPInputConfig, func() error, error) {
				_, key, cleanup, err := createTestCert()
				if err != nil {
					return nil, nil, err
				}

				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.TLS = TLSConfig{
					Enable:      true,
					Certificate: "",
					PrivateKey:  key,
					MinVersion:  1.2,
				}

				return cfg, cleanup, nil
			},
			true,
			"missing required parameter 'certificate', required when TLS is enabled",
		},
		{
			"missing-key-file",
			func() (*HTTPInputConfig, func() error, error) {
				crt, _, cleanup, err := createTestCert()
				if err != nil {
					return nil, nil, err
				}

				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.TLS = TLSConfig{
					Enable:      true,
					Certificate: crt,
					PrivateKey:  "",
					MinVersion:  1.2,
				}

				return cfg, cleanup, nil
			},
			true,
			"missing required parameter 'private_key', required when TLS is enabled",
		},
		{
			"wrong-certificate-path",
			func() (*HTTPInputConfig, func() error, error) {
				_, key, cleanup, err := createTestCert()
				if err != nil {
					return nil, nil, err
				}

				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.TLS = TLSConfig{
					Enable:      true,
					Certificate: "/tmp/some-invalid-path",
					PrivateKey:  key,
					MinVersion:  1.2,
				}

				return cfg, cleanup, nil
			},
			true,
			"failed to load tls certificate",
		},
		{
			"wrong-private-key-path",
			func() (*HTTPInputConfig, func() error, error) {
				crt, _, cleanup, err := createTestCert()
				if err != nil {
					return nil, nil, err
				}

				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.TLS = TLSConfig{
					Enable:      true,
					Certificate: crt,
					PrivateKey:  "/invalid/path",
					MinVersion:  1.2,
				}

				return cfg, cleanup, nil
			},
			true,
			"failed to load tls",
		},
		{
			"read-timeout",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.ReadTimeout = helper.NewDuration(10)
				return cfg, nil, nil
			},
			false,
			"",
		},
		{
			"read-timeout-0",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.ReadTimeout = helper.NewDuration(0)
				return cfg, nil, nil
			},
			false,
			"",
		},
		{
			"read-timeout-negative",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.ReadTimeout = helper.NewDuration(-1)
				return cfg, nil, nil
			},
			true,
			"read_timeout cannot be less than 0",
		},
		{
			"idle-timeout",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.IdleTimeout = helper.NewDuration(10)
				return cfg, nil, nil
			},
			false,
			"",
		},
		{
			"idle-timeout-0",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.IdleTimeout = helper.NewDuration(0)
				return cfg, nil, nil
			},
			false,
			"",
		},
		{
			"idle-timeout-negative",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.IdleTimeout = helper.NewDuration(-1)
				return cfg, nil, nil
			},
			true,
			"idle_timeout cannot be less than 0",
		},
		{
			"write-timeout",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.WriteTimeout = helper.NewDuration(10)
				return cfg, nil, nil
			},
			false,
			"",
		},
		{
			"write-timeout-0",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.WriteTimeout = helper.NewDuration(0)
				return cfg, nil, nil
			},
			false,
			"",
		},
		{
			"write-timeout-negative",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.WriteTimeout = helper.NewDuration(-1)
				return cfg, nil, nil
			},
			true,
			"write_timeout cannot be less than 0",
		},
		{
			"max-header-size",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.MaxHeaderSize = helper.ByteSize(100)
				return cfg, nil, nil
			},
			false,
			"",
		},
		{
			"max-header-size-0",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.MaxHeaderSize = helper.ByteSize(0)
				return cfg, nil, nil
			},
			false,
			"",
		},
		{
			"max-header-negative",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.MaxHeaderSize = helper.ByteSize(-1)
				return cfg, nil, nil
			},
			true,
			"max_header_size cannot be less than 0",
		},
		{
			"max-body-size",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.MaxBodySize = helper.ByteSize(1000)
				return cfg, nil, nil
			},
			false,
			"",
		},
		{
			"max-body-size-0",
			func() (*HTTPInputConfig, func() error, error) {
				cfg := NewHTTPInputConfig("test_id")
				cfg.ListenAddress = defaultListenAddr
				cfg.MaxBodySize = helper.ByteSize(0)
				return cfg, nil, nil
			},
			true,
			"max_body_size cannot be less than 1 byte",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, cleanup, err := tc.input()
			require.NoError(t, err)
			defer func() {
				if cleanup != nil {
					require.NoError(t, cleanup())
				}
			}()

			_, err = cfg.Build(operator.BuildContext{})
			require.Error(t, err, "expected empty build context to cause an error")

			op, err := cfg.build(testutil.NewBuildContext(t))
			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errSubStr)
				return
			}
			require.NoError(t, err)

			require.Equal(t, cfg.ListenAddress, op.server.Addr)
			require.NotEmpty(t, op.server.TLSConfig.MinVersion)
			require.NotEmpty(t, op.server.TLSConfig.Certificates)
			require.Equal(t, cfg.ReadTimeout.Duration, op.server.ReadTimeout)
			require.Equal(t, cfg.ReadTimeout.Duration, op.server.ReadHeaderTimeout)
			require.Equal(t, cfg.WriteTimeout.Duration, op.server.WriteTimeout)
			require.Equal(t, cfg.IdleTimeout.Duration, op.server.IdleTimeout)
			require.Equal(t, int(cfg.MaxHeaderSize), op.server.MaxHeaderBytes)
			require.Equal(t, int64(cfg.MaxBodySize), op.maxBodySize)
		})
	}
}

func TestBuild(t *testing.T) {
	cfg := NewHTTPInputConfig("test_id")
	cfg.ListenAddress = ":0"
	ops, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
	require.Equal(t, 1, len(ops))
}

// createTestCert writes a key pair to the file system and returns
// the certificate's path, private key's path, cleanup function, error
func createTestCert() (string, string, func() error, error) {
	crt, err := ioutil.TempFile("./", "test.crt")
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to open temp file for certificate: %s", err)
	}
	defer crt.Close()
	_, err = crt.WriteString(testTLSCertificate + "\n")
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to write test certificate: %s", err)
	}

	key, err := ioutil.TempFile("./", "test.key")
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to open temp file for private key: %s", err)
	}
	defer key.Close()
	_, err = key.WriteString(testTLSPrivateKey + "\n")
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to write test private key: %s", err)
	}

	cleanup := func() error {
		err1 := os.Remove(crt.Name())
		err2 := os.Remove(key.Name())
		if err1 != nil || err2 != nil {
			return fmt.Errorf("error cleaning up test certificate: %s, %s", err1, err2)
		}
		return nil
	}

	return crt.Name(), key.Name(), cleanup, nil
}

const testTLSPrivateKey = `
-----BEGIN PRIVATE KEY-----
MIIEvwIBADANBgkqhkiG9w0BAQEFAASCBKkwggSlAgEAAoIBAQDdNdVRHDoOlwrQ
YNlzP6MdLEIvN03Pv3A/Cdyy8LgKgSEf3kmw8o/75tSQzIAR6v7ts/qq1iAwE3OL
s4r8lASj2wirF2fNxX12OvIP8g3mrs4tCANBh413IywVKcEOrry71/s1k7+hscMv
Fe3NLxD1mNKJogwKyifvSc15zx8ge8SLjp875NiLCni2YYWXBt1pqd4wCol8lX6v
3u2rbNXrQf2sLncD0CE45EWHnzLzK33a0BwxyTXAOdd9kindL2IFct9C2HRQEk5h
GaXbNN0f6EMOZOzadJHfMledKVJ1XOd+t/kaPzY4NLDaGad04pNa+jph54qIVL5b
gCTOivX1AgMBAAECggEBAKPll/hxrn5S4LtFlrdyJfueaCctlaRgFd1PBEs8WU/H
HvDKtNS6031zKHlkW1trPpiF6iqbXdvg/ZI7Y7YCQXHZ/pEtVUa7lVp9EA5KbIxH
ZhEtR6RMt77Wu3mupxCm3MVcoA6xOqGl4JTJbZjBz5H4Ob2p57wyzeXYS7p9gHWC
fSj8tEqJdjLt7lqtqaWg/3iqqnLPdT3fGL6uyVbCDn9VZ23C7+sHiUfG67xHiF97
UT+O+dfADMY6rLY1njxdD0QGPS7MQLHAgL/ESjROSL4cj1f9VYJFgweAE/UxnDVQ
n3pTzHFItjYWtK75o7Yc/zaHKp5hsXMsiVb9gtmBcaECgYEA+i2viVdZQqItIDiJ
rc7M42Fo6mLv1gToOVaIst7qPmW6BlwSQbX/x2V/2UsMWtcL95mrmRVjK9iH/Pg8
ZaMlJynpgTM/x0jlZ2gZW1DPJWiCJ97xsdbOBA4JiGExc7odkbZhecfdlf66h0N6
Ll32k80PNqTDJV8wWuUxsEnJaLkCgYEA4luVgtnhiJx3FIfBM9p/EVearFsQFSil
PPeoJfc5GMGAnNeGBv5YI4wZ5Jaa0qHLg5ps5Y8vO1yWKiAuhgVKXhytOj86XsoL
MdisDYcxzskG/9ipX3fP1rBNgwdzBoP4QcpzV69weDsja8AU2pluKSd3r3nzwqsY
dc/NVJRsYR0CgYAw2scSrOoTZxQk3KWWOXItXRJd4yAuzRqER++97mYT9U2UfFpc
VqwyRhHnXw50ltYRbgLijBinsUstDVTODEPvF/IvdtCXnBagUOXSvT8WcQgpvRG5
xtbIV+1oooJDtS6dC96RJ4SQDARk8bpkX5kNV9gGtboeDC6nMWa4pFAekQKBgQCm
naM/3gEU/ZbplcOw13QQ39sKYz1DVdfLOMCcsY1lm4l/6WTOYQmfoNCuYe00fcO/
6zuc/fhWSaB/AZE9NUe4XoNkDIZ6n13+Iu8CRjFzdKWiTWjezOI/tSZY/HK+qQVj
6BFeydSPq3g3J/wxrB5aTKLcl3fGIwquLXeGenoMQQKBgQCWULypEeQwJsyKB57P
JzuCnFMvLL5qSNwot5c7I+AX5yi368dEurQl6pUUJ9VKNbpsUxFIMq9AHpddDoq/
+nIVt1DYr55ZsUJ6SgYtjvCMT9WOE/1Kqfh6p6y/mgRUl8m6v6gqi5/RfsNWJwfl
iBXhcGCQfkwZ8YIUyTW89qrwMw==
-----END PRIVATE KEY-----`

const testTLSCertificate = `
-----BEGIN CERTIFICATE-----
MIIDVDCCAjwCCQCwsE+LGRRtBTANBgkqhkiG9w0BAQsFADBsMQswCQYDVQQGEwJV
UzERMA8GA1UECAwITWljaGlnYW4xFTATBgNVBAcMDEdyYW5kIFJhcGlkczERMA8G
A1UECgwIb2JzZXJ2aVExDzANBgNVBAsMBlN0YW56YTEPMA0GA1UEAwwGU3Rhbnph
MB4XDTIxMDIyNTE3MzgxM1oXDTQ4MDcxMjE3MzgxM1owbDELMAkGA1UEBhMCVVMx
ETAPBgNVBAgMCE1pY2hpZ2FuMRUwEwYDVQQHDAxHcmFuZCBSYXBpZHMxETAPBgNV
BAoMCG9ic2VydmlRMQ8wDQYDVQQLDAZTdGFuemExDzANBgNVBAMMBlN0YW56YTCC
ASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAN011VEcOg6XCtBg2XM/ox0s
Qi83Tc+/cD8J3LLwuAqBIR/eSbDyj/vm1JDMgBHq/u2z+qrWIDATc4uzivyUBKPb
CKsXZ83FfXY68g/yDeauzi0IA0GHjXcjLBUpwQ6uvLvX+zWTv6Gxwy8V7c0vEPWY
0omiDArKJ+9JzXnPHyB7xIuOnzvk2IsKeLZhhZcG3Wmp3jAKiXyVfq/e7ats1etB
/awudwPQITjkRYefMvMrfdrQHDHJNcA5132SKd0vYgVy30LYdFASTmEZpds03R/o
Qw5k7Np0kd8yV50pUnVc5363+Ro/Njg0sNoZp3Tik1r6OmHniohUvluAJM6K9fUC
AwEAATANBgkqhkiG9w0BAQsFAAOCAQEA0u061goAXX7RxtdRO7Twz4zZIGS/oWvn
gj61zZIXt8LaTzRZFU9rs0rp7jPXKaszArJQc29anf1mWtRwQBAY0S0m4DkwoBln
7hMFf9MlisQvBVFjWgDo7QCJJmAxaPc1NZi8GQIANEMMZ+hLK17dhDB+6SdBbV4R
yx+7I3zcXQ+0H4Aym6KmvoIR3QAXsOYJ/43QzlYU63ryGYBAeg+JiD8fnr2W3QHb
BBdatHmcazlytT5KV+bANT/Ermw8y2tpWGWxMxQHveFh1zThYL8vkLi4fmZqqVCI
zv9WEy+9p05Aet+12x3dzRu93+yRIEYbSZ35NOUWfQ+gspF5rGgpxA==
-----END CERTIFICATE-----`
