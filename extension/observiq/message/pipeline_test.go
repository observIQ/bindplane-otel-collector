package message

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPipeline(t *testing.T) {
	pipeline := NewPipeline(30)
	require.Equal(t, cap(pipeline.inbound), 30)
	require.Equal(t, cap(pipeline.outbound), 30)
}

func TestPipelineInbound(t *testing.T) {
	pipeline := NewPipeline(30)
	require.Equal(t, pipeline.inbound, pipeline.Inbound())
}

func TestPipelineOutbound(t *testing.T) {
	pipeline := NewPipeline(30)
	require.Equal(t, pipeline.outbound, pipeline.Outbound())
}
