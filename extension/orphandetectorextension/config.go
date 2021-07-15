package orphandetectorextension

import (
	"time"

	"go.opentelemetry.io/collector/config"
)

// Config for configuring the orphan detector extension
type Config struct {
	config.ExtensionSettings `mapstructure:",squash"`
	// Interval is the interval on which the extension checks if it's been orphaned.
	Interval time.Duration `mapstructure:"interval"`
	// Parent process id. Normally, you wouldn't put this in the config, and would use the --set cli option to set this.
	// If the processes parent ID differs from this one during a poll, the extension will report a fatal error to the host.
	// If it is not defined, a ppid is gotten at extension create time using os.Getppid
	Ppid int `mapstructure:"ppid"`
	// If DieOnInitParent is true, the orphandetector extension will fire a fatal error when ppid = 1 (on linux or darwin)
	DieOnInitParent bool `mapstructure:"die_on_init_parent"`
}
