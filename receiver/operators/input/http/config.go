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
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/helper"
)

const (
	defaultTimeout     = time.Second * 20
	defaultIdleTimeout = time.Second * 60
	defaultMaxBodySize = 10000000 // 10 megabyte
)

// NewHTTPInputConfig creates a new HTTP input config with default values
func NewHTTPInputConfig(operatorID string) *HTTPInputConfig {
	return &HTTPInputConfig{
		InputConfig:   helper.NewInputConfig(operatorID, "http_input"),
		IdleTimeout:   helper.NewDuration(defaultIdleTimeout),
		ReadTimeout:   helper.NewDuration(defaultTimeout),
		WriteTimeout:  helper.NewDuration(defaultTimeout),
		MaxHeaderSize: helper.ByteSize(http.DefaultMaxHeaderBytes),
		MaxBodySize:   helper.ByteSize(defaultMaxBodySize),
	}
}

// HTTPInputConfig is the configuration of a http input operator.
type HTTPInputConfig struct {
	helper.InputConfig `yaml:",inline"`

	ListenAddress string          `json:"listen_address,omitempty"  yaml:"listen_address,omitempty"`
	TLS           TLSConfig       `json:"tls,omitempty"             yaml:"tls,omitempty"`
	IdleTimeout   helper.Duration `json:"idle_timeout,omitempty"    yaml:"idle_timeout,omitempty"`
	ReadTimeout   helper.Duration `json:"read_timeout,omitempty"    yaml:"read_timeout,omitempty"`
	WriteTimeout  helper.Duration `json:"write_timeout,omitempty"   yaml:"write_timeout,omitempty"`
	MaxHeaderSize helper.ByteSize `json:"max_header_size,omitempty" yaml:"max_header_size,omitempty"`
	MaxBodySize   helper.ByteSize `json:"max_body_size,omitempty"   yaml:"max_body_size,omitempty"`
	AuthConfig    authConfig      `json:"auth,omitempty"   yaml:"auth,omitempty"`
}

// TLSConfig is the configuration for a TLS listener
type TLSConfig struct {
	// Enable forces the user of TLS
	Enable bool `json:"enable,omitempty" yaml:"enable,omitempty"`

	// Certificate is the file path for the certificate
	Certificate string `json:"certificate,omitempty" yaml:"certificate,omitempty"`

	// PrivateKey is the file path for the private key
	PrivateKey string `json:"private_key,omitempty" yaml:"private_key,omitempty"`

	// MinVersion is the minimum tls version
	MinVersion float32 `json:"min_version,omitempty" yaml:"min_version,omitempty"`
}

type authConfig struct {
	TokenHeader string   `json:"token_header,omitempty" yaml:"token_header,omitempty"`
	Tokens      []string `json:"tokens,omitempty"       yaml:"tokens,omitempty"`
	Username    string   `json:"username,omitempty"       yaml:"username,omitempty"`
	Password    string   `json:"password,omitempty"       yaml:"password,omitempty"`
}

// Build will build a http input operator.
func (c HTTPInputConfig) Build(ctx operator.BuildContext) ([]operator.Operator, error) {
	op, err := c.build(ctx)
	return []operator.Operator{op}, err
}

func (c HTTPInputConfig) build(context operator.BuildContext) (*HTTPInput, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return &HTTPInput{}, err
	}

	if c.ListenAddress == "" {
		return &HTTPInput{}, fmt.Errorf("missing required parameter 'listen_address'")
	}

	// validate the input address
	if _, err := net.ResolveTCPAddr("tcp", c.ListenAddress); err != nil {
		return &HTTPInput{}, fmt.Errorf("failed to resolve listen_address: %s", err)
	}

	cert := tls.Certificate{}
	if c.TLS.Enable {
		if c.TLS.Certificate == "" {
			return &HTTPInput{}, fmt.Errorf("missing required parameter 'certificate', required when TLS is enabled")
		}

		if c.TLS.PrivateKey == "" {
			return &HTTPInput{}, fmt.Errorf("missing required parameter 'private_key', required when TLS is enabled")
		}

		c, err := tls.LoadX509KeyPair(c.TLS.Certificate, c.TLS.PrivateKey)
		if err != nil {
			return &HTTPInput{}, fmt.Errorf("failed to load tls certificate: %w", err)
		}
		cert = c
	}

	// Allow user to configure 0 for timeout values as this is the default behavior
	if c.IdleTimeout.Seconds() < 0 {
		return &HTTPInput{}, fmt.Errorf("idle_timeout cannot be less than 0")
	}
	if c.ReadTimeout.Seconds() < 0 {
		return &HTTPInput{}, fmt.Errorf("read_timeout cannot be less than 0")
	}
	if c.WriteTimeout.Seconds() < 0 {
		return &HTTPInput{}, fmt.Errorf("write_timeout cannot be less than 0")
	}

	// Allow user to configure 0 for max header size as this is the default behavior
	if c.MaxHeaderSize < 0 {
		return &HTTPInput{}, fmt.Errorf("max_header_size cannot be less than 0")
	}

	if c.MaxBodySize < 1 {
		return &HTTPInput{}, fmt.Errorf("max_body_size cannot be less than 1 byte")
	}

	var tlsMinVersion uint16
	switch c.TLS.MinVersion {
	case 1.0:
		tlsMinVersion = tls.VersionTLS10
	case 1.1:
		tlsMinVersion = tls.VersionTLS11

	// TLS 1.0 is the default version implemented by cypto/tls https://pkg.go.dev/crypto/tls#Config
	// however this operator will default to TLS 1.2 when tls version is not set.
	case 0, 1.2:
		tlsMinVersion = tls.VersionTLS12
	case 1.3:
		tlsMinVersion = tls.VersionTLS13
	default:
		return &HTTPInput{}, fmt.Errorf("unsupported tls version: %f", c.TLS.MinVersion)
	}

	if c.AuthConfig.TokenHeader != "" && c.AuthConfig.Username != "" {
		return &HTTPInput{}, fmt.Errorf("token auth and basic auth cannot be enabled at the same time")
	}

	if c.AuthConfig.Username != "" && c.AuthConfig.Password == "" {
		return &HTTPInput{}, fmt.Errorf("password must be set when basic auth username is set")
	}

	if c.AuthConfig.Password != "" && c.AuthConfig.Username == "" {
		return &HTTPInput{}, fmt.Errorf("username must be set when basic auth password is set")
	}

	if c.AuthConfig.TokenHeader != "" {
		if len(c.AuthConfig.Tokens) == 0 {
			return &HTTPInput{}, fmt.Errorf("auth.tokens is a required parameter when auth.token_header is set")
		}
	}

	var auth authMiddleware
	if c.AuthConfig.TokenHeader != "" {
		auth = authToken{
			tokenHeader: c.AuthConfig.TokenHeader,
			tokens:      c.AuthConfig.Tokens,
		}
	} else if c.AuthConfig.Username != "" {
		auth = authBasic{
			username: c.AuthConfig.Username,
			password: c.AuthConfig.Password,
		}
	}

	httpInput := &HTTPInput{
		InputOperator: inputOperator,
		tls:           c.TLS.Enable,
		server: http.Server{
			Addr: c.ListenAddress,
			// #nosec - User to specify tls minimum version
			TLSConfig: &tls.Config{
				MinVersion:   tlsMinVersion,
				Certificates: []tls.Certificate{cert},
			},
			ReadTimeout:       c.ReadTimeout.Raw(),
			ReadHeaderTimeout: c.ReadTimeout.Raw(),
			WriteTimeout:      c.WriteTimeout.Raw(),
			IdleTimeout:       c.IdleTimeout.Raw(),
			/*
				 This value is padded with 4096 bytes
				 https://cs.opensource.google/go/go/+/refs/tags/go1.17.1:src/net/http/server.go;l=865

				func (srv *Server) initialReadLimitSize() int64 {
					return int64(srv.maxHeaderBytes()) + 4096 // bufio slop
				}
			*/
			MaxHeaderBytes: int(c.MaxHeaderSize),
			TLSNextProto:   nil, // This should be configured if we want HTTP/2 support
			ConnState:      nil,
			ErrorLog:       nil, // TODO: logger logs http server errors
			BaseContext:    nil,
			ConnContext:    nil,
		},
		maxBodySize: int64(c.MaxBodySize),
		json:        jsoniter.ConfigFastest,
		auth:        auth,
	}

	return httpInput, nil
}
