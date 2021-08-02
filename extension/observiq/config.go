package observiq

import (
	"time"

	"go.opentelemetry.io/collector/config"
)

const (
	typeStr           = "observiq_controller"
	endpoint          = "wss://connections.app.observiq.com"
	statusInterval    = time.Second * 5
	reconnectInterval = time.Minute * 30
)

// Config is the configuration of an observiq extension
type Config struct {
	Endpoint          string        `mapstructure:"endpoint"`
	AgentName         string        `mapstructure:"agent_name"`
	AgentID           string        `mapstructure:"agent_id"`
	SecretKey         string        `mapstructure:"secret_key"`
	StatusInterval    time.Duration `mapstructure:"status_interval"`
	ReconnectInterval time.Duration `mapstructure:"reconnect_interval"`
	TemplateID        string        `mapstructure:"template_id"`

	config.ExtensionSettings `mapstructure:",squash"`
}

// createDefaultConfig returns the default config used to configure the observiq extension
func createDefaultConfig() config.Extension {
	return &Config{
		ExtensionSettings: config.NewExtensionSettings(config.NewID(typeStr)),
		Endpoint:          endpoint,
		StatusInterval:    statusInterval,
		ReconnectInterval: reconnectInterval,
	}
}
