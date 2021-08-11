package status

import (
	"testing"

	"github.com/observiq/observiq-collector/manager/message"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	report := Get()
	require.Equal(t, report.ComponentID, "bpagent")
}

func TestReportToMessage(t *testing.T) {
	report := Get()

	msg := report.ToMessage()
	require.Equal(t, msg.Type, message.StatusReport)
}
