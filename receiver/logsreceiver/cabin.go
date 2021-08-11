package logsreceiver

import (
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/noop"
)

func init() {
	operator.Register("cabin_output", func() operator.Builder { return noop.NewNoopOperatorConfig("") })
}
