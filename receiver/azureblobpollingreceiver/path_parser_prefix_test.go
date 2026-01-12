package azureblobpollingreceiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateTimePrefixes(t *testing.T) {
	tests := []struct {
		name       string
		startTime  time.Time
		endTime    time.Time
		pattern    string
		rootFolder string
		expected   []string
		expectErr  bool
	}{
		{
			name:       "Standard hourly pattern",
			startTime:  time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			endTime:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			pattern:    "{year}/{month}/{day}/{hour}",
			rootFolder: "",
			expected: []string{
				"2025/01/01/10",
				"2025/01/01/11",
				"2025/01/01/12",
			},
		},
		{
			name:       "Hourly pattern with root folder",
			startTime:  time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			endTime:    time.Date(2025, 1, 1, 10, 30, 0, 0, time.UTC),
			pattern:    "{year}/{month}/{day}/{hour}",
			rootFolder: "logs",
			expected: []string{
				"logs/2025/01/01/10",
			},
		},
		{
			name:       "Pattern with minutes (truncated to hour)",
			startTime:  time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			endTime:    time.Date(2025, 1, 1, 11, 0, 0, 0, time.UTC),
			pattern:    "{year}/{month}/{day}/{hour}/{minute}",
			rootFolder: "",
			expected: []string{
				"2025/01/01/10",
				"2025/01/01/11",
			},
		},
		{
			name:       "Daily pattern (fallback to day)",
			startTime:  time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			endTime:    time.Date(2025, 1, 3, 10, 0, 0, 0, time.UTC),
			pattern:    "{year}/{month}/{day}",
			rootFolder: "",
			expected: []string{
				"2025/01/01",
				"2025/01/02",
				"2025/01/03",
			},
		},
		{
			name:       "Go time format",
			startTime:  time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			endTime:    time.Date(2025, 1, 1, 11, 0, 0, 0, time.UTC),
			pattern:    "2006/01/02/15",
			rootFolder: "data",
			expected: []string{
				"data/2025/01/01/10",
				"data/2025/01/01/11",
			},
		},
		{
			name:       "Root folder with trailing slash",
			startTime:  time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			endTime:    time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC),
			pattern:    "{year}",
			rootFolder: "logs/",
			expected: []string{
				"logs/2025",
			},
		},
		{
			name:       "Crossing hour boundary (9:55 to 10:05)",
			startTime:  time.Date(2025, 1, 1, 9, 55, 0, 0, time.UTC),
			endTime:    time.Date(2025, 1, 1, 10, 5, 0, 0, time.UTC),
			pattern:    "{year}/{month}/{day}/{hour}",
			rootFolder: "",
			expected: []string{
				"2025/01/01/09",
				"2025/01/01/10",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefixes, err := GenerateTimePrefixes(tt.startTime, tt.endTime, tt.pattern, tt.rootFolder)
			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, prefixes)
			}
		})
	}
}
