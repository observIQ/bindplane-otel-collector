package status

import (
	"testing"

	"github.com/observiq/observiq-collector/extension/observiq/message"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	report, err := Get()
	require.NoError(t, err)
	require.Equal(t, report.ComponentID, "bpagent")
}

func TestReportToMessage(t *testing.T) {
	report, err := Get()
	require.NoError(t, err)

	msg := report.ToMessage()
	require.Equal(t, msg.Type, message.StatusReport)
}
