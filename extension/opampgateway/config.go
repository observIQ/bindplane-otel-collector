package opampgateway

type Config struct {
	UpstreamOpAMPAddress string       `mapstructure:"upstream_opamp_address"`
	SecretKey            string       `mapstructure:"secret_key"`
	UpstreamConnections  int          `mapstructure:"upstream_connections"`
	OpAMPServer          *OpAMPServer `mapstructure:"opamp_server"`
}

type OpAMPServer struct {
	Endpoint string     `mapstructure:"endpoint"`
	TLS      *TLSConfig `mapstructure:"tls"`
}

type TLSConfig struct {
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
	KeyFile            string `mapstructure:"key_file"`
	CertFile           string `mapstructure:"cert_file"`
}
