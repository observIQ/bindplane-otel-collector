package opampgateway

import (
	"crypto/tls"
	"errors"
	"net/url"
)

type Config struct {
	UpstreamOpAMPAddress string       `mapstructure:"upstream_opamp_address"`
	SecretKey            string       `mapstructure:"secret_key"`
	UpstreamConnections  int          `mapstructure:"upstream_connections"`
	OpAMPServer          *OpAMPServer `mapstructure:"opamp_server"`
}

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	if c.UpstreamOpAMPAddress == "" {
		return errors.New("upstream_opamp_address must be specified")
	}
	u, err := url.Parse(c.UpstreamOpAMPAddress)
	if err != nil {
		return errors.New("upstream_opamp_address is not a valid URL")
	}
	if u.Scheme != "ws" && u.Scheme != "wss" {
		return errors.New("upstream_opamp_address must use ws:// or wss:// scheme")
	}

	if c.UpstreamConnections < 1 {
		return errors.New("upstream_connections must be at least 1")
	}

	if c.OpAMPServer == nil {
		return errors.New("opamp_server must be specified")
	}
	if c.OpAMPServer.Endpoint == "" {
		return errors.New("opamp_server endpoint must be specified")
	}

	return nil
}

type OpAMPServer struct {
	Endpoint string      `mapstructure:"endpoint"`
	TLS      *tls.Config `mapstructure:",squash"`
}
