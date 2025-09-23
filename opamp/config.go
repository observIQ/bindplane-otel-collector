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
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/open-telemetry/opamp-go/client/types"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
)

var (
	// errPrefixResolverInitialization for error when initializing config file resolver
	errPrefixResolverInitialization = "failed to initialize OpAmp config resolver"

	// errPrefixReadFile for error when reading config file
	errPrefixReadFile = "failed to read OpAmp config file"

	// errPrefixParse for error when parsing config
	errPrefixParse = "failed to parse OpAmp config"

	// errMissingTLSFiles is the error when only one of Key or Cert is specified
	errMissingTLSFiles = "must specify both Key and Certificate file"

	// errInvalidKeyFile for key file that is not readable
	errInvalidKeyFile = "failed to read TLS key file"

	// errInvalidCertFile for cert file that is not readable
	errInvalidCertFile = "failed to read TLS cert file"

	// errInvalidCAFile for ca file that is not readable
	errInvalidCAFile = "failed to read TLS CA file"
)

type agentIDType string

const (
	agentIDTypeULID agentIDType = "ULID"
	agentIDTypeUUID agentIDType = "UUID"
)

// AgentID represents the ID of the agent
type AgentID struct {
	by     [16]byte
	idType agentIDType
	orig   string
}

// EmptyAgentID represents an empty/unset agent ID.
var EmptyAgentID = AgentID{}

// ParseAgentID parses an agent ID from the given string
func ParseAgentID(s string) (AgentID, error) {
	switch len(s) {
	case 26:
		// length 26 strings can't be UUID, so they must be ULID
		u, err := ulid.Parse(s)
		if err != nil {
			return AgentID{}, fmt.Errorf("parse ulid: %w", err)
		}
		return AgentID{
			by:     u,
			idType: agentIDTypeULID,
			orig:   s,
		}, nil

	default:
		// Try parsing as a UUID. There are a couple forms of UUID supported for parsing, so they may be a couple different
		// lengths
		u, err := uuid.Parse(s)
		if err != nil {
			return AgentID{}, fmt.Errorf("parse uuid: %w", err)
		}
		return AgentID{
			by:     u,
			idType: agentIDTypeUUID,
			orig:   s,
		}, nil
	}
}

// AgentIDFromUUID creates an agent ID from a generated UUID.
// See ParseAgentID for parsing a UUID string.
func AgentIDFromUUID(u uuid.UUID) AgentID {
	return AgentID{
		by:     u,
		idType: agentIDTypeUUID,
		orig:   u.String(),
	}
}

// String returns a string representation of the agent ID
func (a AgentID) String() string {
	return a.orig
}

// OpAMPInstanceUID returns the opamp representation of the agent ID
func (a AgentID) OpAMPInstanceUID() types.InstanceUid {
	return types.InstanceUid(a.by)
}

// Type returns the string type of the agent ID (ULID, UUID) as it should
// be reported to Bindplane.
func (a AgentID) Type() string {
	return string(a.idType)
}

// MarshalYAML implements the yaml.Marshaler interface
func (a AgentID) MarshalYAML() (any, error) {
	return a.String(), nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface
func (a *AgentID) UnmarshalYAML(unmarshal func(any) error) error {
	var s string

	err := unmarshal(&s)
	if err != nil {
		return err
	}

	if s == "" {
		// Empty string = keep the 0 value
		return nil
	}

	u, err := ParseAgentID(s)
	if err != nil {
		// In order to maintain backwards compatability, we don't error here.
		// Instead, in main, we will regenerate a new agent ID
		*a = EmptyAgentID
		return nil
	}

	*a = AgentID(u)

	return nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (a *AgentID) UnmarshalText(text []byte) error {
	s := string(text)

	if s == "" {
		// Empty string = keep the 0 value
		return nil
	}

	u, err := ParseAgentID(s)
	if err != nil {
		// In order to maintain backwards compatability, we don't error here.
		// Instead, in main, we will regenerate a new agent ID
		*a = EmptyAgentID
		return nil
	}

	*a = AgentID(u)

	return nil
}

// MarshalText implements the encoding.TextMarshaler interface
func (a *AgentID) MarshalText() ([]byte, error) {
	// Format the time in your desired format
	return []byte(a.String()), nil
}

// Config contains the configuration for the collector to communicate with an OpAmp enabled platform.
type Config struct {
	Endpoint   string     `yaml:"endpoint" mapstructure:"endpoint"`
	SecretKey  *string    `yaml:"secret_key,omitempty" mapstructure:"secret_key,omitempty"`
	AgentID    AgentID    `yaml:"agent_id" mapstructure:"agent_id"`
	TLS        *TLSConfig `yaml:"tls_config,omitempty" mapstructure:"tls_config,omitempty"`
	InstanceID string     `yaml:"-,omitempty" mapstructure:"-,omitempty"` // This is not in the config file, but is set internally by the client

	// Updatable fields
	Labels                      *string           `yaml:"labels,omitempty" mapstructure:"labels,omitempty"`
	AgentName                   *string           `yaml:"agent_name,omitempty" mapstructure:"agent_name,omitempty"`
	MeasurementsInterval        time.Duration     `yaml:"measurements_interval,omitempty" mapstructure:"measurements_interval,omitempty"`
	ExtraMeasurementsAttributes map[string]string `yaml:"extra_measurements_attributes,omitempty" mapstructure:"extra_measurements_attributes,omitempty"`
	TopologyInterval            *time.Duration    `yaml:"topology_interval" mapstructure:"topology_interval"`
}

// TLSConfig represents the TLS config to connect to OpAmp server
type TLSConfig struct {
	InsecureSkipVerify bool    `yaml:"insecure_skip_verify" mapstructure:"insecure_skip_verify"`
	KeyFile            *string `yaml:"key_file" mapstructure:"key_file"`
	CertFile           *string `yaml:"cert_file" mapstructure:"cert_file"`
	CAFile             *string `yaml:"ca_file" mapstructure:"ca_file"`
}

// ToTLS converts the config to a tls.Config
func (c Config) ToTLS(caCertPool *x509.CertPool) (*tls.Config, error) {
	if c.TLS == nil {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if c.TLS.InsecureSkipVerify {
		tlsConfig.InsecureSkipVerify = true
	}

	// Load CA cert if specified
	if c.TLS.CAFile != nil {
		caCert, err := os.ReadFile(*c.TLS.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}
		if caCertPool == nil {
			caCertPool = x509.NewCertPool()
		}
		caCertPool.AppendCertsFromPEM(caCert)

		tlsConfig.RootCAs = caCertPool
	}

	// Load cert and key file if specified
	if c.TLS.CertFile != nil && c.TLS.KeyFile != nil {
		cert, err := tls.LoadX509KeyPair(*c.TLS.CertFile, *c.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read Key and Cert file: %w", err)
		}

		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// ParseConfig given a configuration file location will parse the config
func ParseConfig(configLocation string) (*Config, error) {
	configPath := filepath.Clean(configLocation)

	resolverSettings := confmap.ResolverSettings{
		URIs: []string{configPath},
		ProviderFactories: []confmap.ProviderFactory{
			fileprovider.NewFactory(),
			envprovider.NewFactory(),
		},
		ConverterFactories: []confmap.ConverterFactory{},
		DefaultScheme:      "env",
	}

	resolver, err := confmap.NewResolver(resolverSettings)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errPrefixResolverInitialization, err)
	}

	conf, err := resolver.Resolve(context.Background())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errPrefixReadFile, err)
	}

	var config Config

	if err = conf.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("%s: %w", errPrefixParse, err)
	}

	config.InstanceID = ulid.Make().String()
	config.ExtraMeasurementsAttributes = make(map[string]string)
	config.ExtraMeasurementsAttributes["bindplane_instance_id"] = config.InstanceID

	// Using Secure TLS check files
	if config.TLS != nil && !config.TLS.InsecureSkipVerify {
		// If CA file is specified
		if config.TLS.CAFile != nil {
			// Validate CA file exists on disk
			if _, err := os.Stat(*config.TLS.CAFile); errors.Is(err, os.ErrNotExist) {
				return nil, errors.New(errInvalidCAFile)
			}
		}

		switch {
		case config.TLS.CertFile == nil && config.TLS.KeyFile == nil: // Not using mTLS
			// Nothing to do. This case exists to make it easier to check all happy permutations for Key and Cert files
		case config.TLS.CertFile != nil && config.TLS.KeyFile != nil: // Validate both files exist
			if _, err := os.Stat(*config.TLS.KeyFile); errors.Is(err, os.ErrNotExist) {
				return nil, errors.New(errInvalidKeyFile)
			}

			if _, err := os.Stat(*config.TLS.CertFile); errors.Is(err, os.ErrNotExist) {
				return nil, errors.New(errInvalidCertFile)
			}
		default: // Case with only one file is specified
			return nil, errors.New(errMissingTLSFiles)
		}
	}
	return &config, nil
}

// Copy creates a deep copy of this config
func (c Config) Copy() *Config {

	cfgCopy := &Config{
		Endpoint:             c.Endpoint,
		AgentID:              c.AgentID,
		MeasurementsInterval: c.MeasurementsInterval,
		InstanceID:           ulid.Make().String(),
	}

	if c.SecretKey != nil {
		cfgCopy.SecretKey = new(string)
		*cfgCopy.SecretKey = *c.SecretKey
	}
	if c.Labels != nil {
		cfgCopy.Labels = new(string)
		*cfgCopy.Labels = *c.Labels
	}
	if c.AgentName != nil {
		cfgCopy.AgentName = new(string)
		*cfgCopy.AgentName = *c.AgentName
	}
	if c.TLS != nil {
		cfgCopy.TLS = c.TLS.copy()
	}

	// Copy extra measurements attributes
	if c.ExtraMeasurementsAttributes != nil {
		cfgCopy.ExtraMeasurementsAttributes = maps.Clone(c.ExtraMeasurementsAttributes)
	} else {
		cfgCopy.ExtraMeasurementsAttributes = make(map[string]string)
	}
	cfgCopy.ExtraMeasurementsAttributes["bindplane_instance_id"] = cfgCopy.InstanceID

	if c.TopologyInterval != nil {
		cfgCopy.TopologyInterval = new(time.Duration)
		*cfgCopy.TopologyInterval = *c.TopologyInterval
	}
	return cfgCopy
}

func (t TLSConfig) copy() *TLSConfig {
	tlsCopy := TLSConfig{
		InsecureSkipVerify: t.InsecureSkipVerify,
	}

	if t.CertFile != nil {
		tlsCopy.CertFile = new(string)
		*tlsCopy.CertFile = *t.CertFile
	}
	if t.KeyFile != nil {
		tlsCopy.KeyFile = new(string)
		*tlsCopy.KeyFile = *t.KeyFile
	}
	if t.CAFile != nil {
		tlsCopy.CAFile = new(string)
		*tlsCopy.CAFile = *t.CAFile
	}

	return &tlsCopy
}

// GetSecretKey returns secret key if set else returns empty string
func (c Config) GetSecretKey() string {
	if c.SecretKey == nil {
		return ""
	}

	return *c.SecretKey
}

// CmpUpdatableFields compares updatable fields for equality
func (c Config) CmpUpdatableFields(o Config) (equal bool) {
	if !CmpStringPtr(c.AgentName, o.AgentName) {
		return false
	}

	if c.MeasurementsInterval != o.MeasurementsInterval {
		return false
	}

	if !CmpDurationPtr(c.TopologyInterval, o.TopologyInterval) {
		return false
	}

	if !maps.Equal(c.ExtraMeasurementsAttributes, o.ExtraMeasurementsAttributes) {
		return false
	}

	return CmpStringPtr(c.Labels, o.Labels)
}

// CmpStringPtr compares two string pointers for equality
func CmpStringPtr(p1, p2 *string) bool {
	switch {
	case p1 == nil && p2 == nil:
		return true
	case p1 == nil && p2 != nil:
		fallthrough
	case p1 != nil && p2 == nil:
		fallthrough
	case *p1 != *p2:
		return false
	default:
		return true
	}
}

// CmpDurationPtr compares two duration pointers for equality
func CmpDurationPtr(p1, p2 *time.Duration) bool {
	switch {
	case p1 == nil && p2 == nil:
		return true
	case p1 == nil && p2 != nil:
		fallthrough
	case p1 != nil && p2 == nil:
		fallthrough
	case *p1 != *p2:
		return false
	default:
		return true
	}
}
