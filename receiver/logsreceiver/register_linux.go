package logsreceiver

import (
	// Load linux only packages when importing input operators
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/input/journald"
)
