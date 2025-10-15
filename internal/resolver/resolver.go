// Resolver provides utilities for configuring the global DNS
// resolver.
package resolver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"go.uber.org/zap"
)

const (
	// ENVEnable is the environment variable name for enabling the resolver
	ENVEnable = "BINDPLANE_OTEL_COLLECTOR_RESOLVER_ENABLE"
	// ENVServer is the environment variable name for the DNS server address
	ENVServer = "BINDPLANE_OTEL_COLLECTOR_RESOLVER_SERVER"
	// ENVTimeout is the environment variable name for the DNS timeout duration
	ENVTimeout = "BINDPLANE_OTEL_COLLECTOR_RESOLVER_TIMEOUT"
)

// Resolver represents a DNS resolver configuration that can be used to configure
// the global DNS resolver with custom server settings and logging capabilities.
type Resolver struct {
	// Server is the DNS server address in "host:port" format
	Server string `yaml:"server"`
	// Timeout is the timeout duration for DNS queries
	Timeout time.Duration `yaml:"timeout"`

	logger *zap.Logger
}

// New creates a new resolver. It takes a logger and an optional config path. If the
// configPath is set, it will be used to load the resolver configuration.
// Any errors that occur while loading the configuration will be returned.
// If the configPath is not set, it will be configured from the environment
// if the environment variables are set. Any errors that occur while loading
// the configuration from the environment will be returned.
func New(logger *zap.Logger, configPath string) (*Resolver, error) {
	var resolver Resolver
	var err error

	if logger == nil {
		return nil, errors.New("logger is required")
	}

	if configPath != "" {
		resolver, err = loadFromFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load resolver config from file %s: %w", configPath, err)
		}
	} else {
		resolver, err = loadFromEnvironment()
		if err != nil {
			return nil, fmt.Errorf("failed to load resolver config from environment: %w", err)
		}
	}

	// Validate the loaded configuration
	if err := resolver.Validate(); err != nil {
		return nil, fmt.Errorf("invalid resolver configuration: %w", err)
	}

	resolver.logger = logger.Named("resolver").With(
		zap.String("server", resolver.Server),
		zap.Float64("timeout_seconds", resolver.Timeout.Seconds()),
	)

	return &resolver, nil
}

// Validate validates the resolver configuration
func (r *Resolver) Validate() error {
	if r.Server == "" {
		return errors.New("server is required")
	}

	if r.Timeout <= 0 {
		return errors.New("timeout is required and must be greater than 0")
	}

	host, port, err := net.SplitHostPort(r.Server)
	if err != nil {
		return fmt.Errorf("failed to split host and port: %w", err)
	}

	if host == "" {
		return errors.New("host is required")
	}

	if port == "" {
		return errors.New("port is required")
	}

	if net.ParseIP(host) == nil {
		return fmt.Errorf("invalid IP address: %s", host)
	}

	portInt, err := strconv.ParseInt(port, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid port: %s", port)
	}

	if portInt < 0 || portInt > 65535 {
		return fmt.Errorf("port must be between 0 and 65535: %d", portInt)
	}

	return nil
}

// Configure configures the global DNS resolver to use the configured Resolver options.
// It starts with the current default resolver and overrides only the server and timeout
// settings, preserving all other resolver options. This affects all DNS lookups performed
// by the application.
//
// The caller should take care to call Configure during application startup, before network
// requests are made.
func (r *Resolver) Configure() error {
	// Store the current default resolver to preserve its settings
	currentResolver := net.DefaultResolver

	net.DefaultResolver = &net.Resolver{
		// Force use of Go's DNS resolver with the custom Dial. When CGO is disabled,
		// this is the default behavior.
		PreferGo:     true,
		StrictErrors: currentResolver.StrictErrors,

		// "address" is the origonal DNS server address that is overloaded
		// by the user configured DNS server.
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: r.Timeout,
			}

			conn, err := d.DialContext(ctx, network, r.Server)
			if err != nil {
				r.logger.Debug("DNS resolver dial failed",
					zap.String("network", network),
					zap.String("original_server", address),
					zap.Error(err),
				)
			} else {
				r.logger.Debug("DNS resolver dial successful",
					zap.String("network", network),
					zap.String("original_server", address),
				)
			}

			return conn, err
		},
	}

	return nil
}
