package status

import (
	"testing"

	"github.com/observiq/observiq-collector/extension/observiq/message"
	"github.com/stretchr/testify/require"
)

func TestPump(t *testing.T) {
	pipeline := message.NewPipeline(10)
	err := Pump(pipeline)
	require.NoError(t, err)

	report := <-pipeline.Outbound()
	require.Equal(t, "statusReport", report.Type)
}
