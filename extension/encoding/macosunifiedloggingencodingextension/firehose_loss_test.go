package macosunifiedloggingencodingextension

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TODO: consolidate this test with the firehose test file after merge
func TestParseFirehoseLoss(t *testing.T) {
	parsedFirehoseLoss, _ := ParseFirehoseLoss(firehoseLossTestData)
	require.Equal(t, uint64(707475528), parsedFirehoseLoss.StartTime)
	require.Equal(t, uint64(3144863719), parsedFirehoseLoss.EndTime)
	require.Equal(t, uint64(63), parsedFirehoseLoss.Count)
}

var firehoseLossTestData = []byte{72, 56, 43, 42, 0, 0, 0, 0, 231, 207, 114, 187, 0, 0, 0, 0, 63, 0, 0, 0, 0, 0, 0, 0}
