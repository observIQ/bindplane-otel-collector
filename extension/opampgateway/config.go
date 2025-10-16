package opampgateway

import "crypto/tls"

type Config struct {
	UpstreamOpAMPAddress string       `mapstructure:"upstream_opamp_address"`
	SecretKey            string       `mapstructure:"secret_key"`
	UpstreamConnections  int          `mapstructure:"upstream_connections"`
	OpAMPServer          *OpAMPServer `mapstructure:"opamp_server"`
}

type OpAMPServer struct {
	Endpoint string      `mapstructure:"endpoint"`
	TLS      *tls.Config `mapstructure:",squash"`
}
