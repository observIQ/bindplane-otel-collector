// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package blobconsume //import "github.com/observiq/bindplane-otel-collector/internal/blobconsume"

import (
	"bytes"
	"compress/gzip"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pipeline"
)

func TestParseEntityPath(t *testing.T) {
	expectedTimeMinute := time.Date(2023, time.January, 04, 12, 02, 0, 0, time.UTC)
	expectedTimeHour := time.Date(2023, time.January, 04, 12, 00, 0, 0, time.UTC)

	testcases := []struct {
		desc         string
		entityName   string
		expectedTime *time.Time
		expectedType pipeline.Signal
		expectedErr  error
	}{
		{
			desc:         "Empty entityName",
			entityName:   "",
			expectedTime: nil,
			expectedType: pipeline.Signal{},
			expectedErr:  ErrInvalidEntityPath,
		},
		{
			desc:         "Malformed path",
			entityName:   "year=2023/day=04/hour=12/minute=02/entitymetrics_12345.json",
			expectedTime: nil,
			expectedType: pipeline.Signal{},
			expectedErr:  ErrInvalidEntityPath,
		},
		{
			desc:         "Malformed timestamp",
			entityName:   "year=2003/month=00/day=04/hour=12/minute=01/entitymetrics_12345.json",
			expectedTime: nil,
			expectedType: pipeline.Signal{},
			expectedErr:  errors.New("parse entity time"),
		},
		{
			desc:         "Prefix, minute, metrics",
			entityName:   "prefix/year=2023/month=01/day=04/hour=12/minute=02/entitymetrics_12345.json",
			expectedTime: &expectedTimeMinute,
			expectedType: pipeline.SignalMetrics,
			expectedErr:  nil,
		},
		{
			desc:         "No Prefix, minute, metrics",
			entityName:   "year=2023/month=01/day=04/hour=12/minute=02/entitymetrics_12345.json",
			expectedTime: &expectedTimeMinute,
			expectedType: pipeline.SignalMetrics,
			expectedErr:  nil,
		},
		{
			desc:         "No Prefix, minute, logs",
			entityName:   "year=2023/month=01/day=04/hour=12/minute=02/entitylogs_12345.json",
			expectedTime: &expectedTimeMinute,
			expectedType: pipeline.SignalLogs,
			expectedErr:  nil,
		},
		{
			desc:         "No Prefix, minute, traces",
			entityName:   "year=2023/month=01/day=04/hour=12/minute=02/entitytraces_12345.json",
			expectedTime: &expectedTimeMinute,
			expectedType: pipeline.SignalTraces,
			expectedErr:  nil,
		},
		{
			desc:         "No Prefix, hour, metrics",
			entityName:   "year=2023/month=01/day=04/hour=12/entitymetrics_12345.json",
			expectedTime: &expectedTimeHour,
			expectedType: pipeline.SignalMetrics,
			expectedErr:  nil,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			actualTime, actualType, err := ParseEntityPath(tc.entityName)
			if tc.expectedErr != nil {
				require.ErrorContains(t, err, tc.expectedErr.Error())
				require.Nil(t, tc.expectedTime)
			} else {
				require.NoError(t, err)
				require.NotNil(t, actualTime)
				require.True(t, tc.expectedTime.Equal(*actualTime))
				require.Equal(t, tc.expectedType, actualType)
			}
		})
	}
}

func TestIsInTimeRange(t *testing.T) {
	testcases := []struct {
		name         string
		entityTime   time.Time
		startingTime time.Time
		endingTime   time.Time
		expected     bool
	}{
		{
			name:         "Entity time is equal to starting time",
			entityTime:   time.Date(2024, time.January, 01, 12, 00, 00, 00, time.UTC),
			startingTime: time.Date(2024, time.January, 01, 12, 00, 00, 00, time.UTC),
			endingTime:   time.Date(2024, time.January, 01, 13, 00, 00, 00, time.UTC),
			expected:     true,
		},
		{
			name:         "Entity time is equal to ending time",
			entityTime:   time.Date(2024, time.January, 01, 13, 00, 00, 00, time.UTC),
			startingTime: time.Date(2024, time.January, 01, 12, 00, 00, 00, time.UTC),
			endingTime:   time.Date(2024, time.January, 01, 13, 00, 00, 00, time.UTC),
			expected:     true,
		},
		{
			name:         "Entity time is between starting and ending time",
			entityTime:   time.Date(2024, time.January, 01, 12, 30, 00, 00, time.UTC),
			startingTime: time.Date(2024, time.January, 01, 12, 00, 00, 00, time.UTC),
			endingTime:   time.Date(2024, time.January, 01, 13, 00, 00, 00, time.UTC),
			expected:     true,
		},
		{
			name:         "Entity time is before starting time",
			entityTime:   time.Date(2024, time.January, 01, 11, 00, 00, 00, time.UTC),
			startingTime: time.Date(2024, time.January, 01, 12, 00, 00, 00, time.UTC),
			endingTime:   time.Date(2024, time.January, 01, 13, 00, 00, 00, time.UTC),
			expected:     false,
		},
		{
			name:         "Entity time is after ending time",
			entityTime:   time.Date(2024, time.January, 01, 14, 00, 00, 00, time.UTC),
			startingTime: time.Date(2024, time.January, 01, 12, 00, 00, 00, time.UTC),
			endingTime:   time.Date(2024, time.January, 01, 13, 00, 00, 00, time.UTC),
			expected:     false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual := IsInTimeRange(tc.entityTime, tc.startingTime, tc.endingTime)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestGzipDecompress(t *testing.T) {
	raw := []byte{31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 0, 99}

	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	_, err := writer.Write(raw)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	result, err := GzipDecompress(buf.Bytes())
	require.NoError(t, err)
	require.Equal(t, raw, result)
}
