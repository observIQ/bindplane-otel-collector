// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package azureblobpollingreceiver //import "github.com/observiq/bindplane-otel-collector/receiver/azureblobpollingreceiver"

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseTimeFromPattern(t *testing.T) {
	t.Run("Placeholder pattern - full datetime", func(t *testing.T) {
		pattern := "{year}/{month}/{day}/{hour}/{minute}/{second}/data.json"
		blobPath := "2024/03/15/14/30/45/data.json"

		parsedTime, err := parseTimeFromPattern(blobPath, pattern)
		require.NoError(t, err)
		require.NotNil(t, parsedTime)

		expected := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)
		require.Equal(t, expected, *parsedTime)
	})

	t.Run("Placeholder pattern - date only", func(t *testing.T) {
		pattern := "{year}/{month}/{day}/data.json"
		blobPath := "2024/03/15/data.json"

		parsedTime, err := parseTimeFromPattern(blobPath, pattern)
		require.NoError(t, err)
		require.NotNil(t, parsedTime)

		expected := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
		require.Equal(t, expected, *parsedTime)
	})

	t.Run("Placeholder pattern - with prefix", func(t *testing.T) {
		pattern := "logs/{year}/{month}/{day}/file.json"
		blobPath := "logs/2024/03/15/file.json"

		parsedTime, err := parseTimeFromPattern(blobPath, pattern)
		require.NoError(t, err)
		require.NotNil(t, parsedTime)

		expected := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
		require.Equal(t, expected, *parsedTime)
	})

	t.Run("Placeholder pattern - year and month only", func(t *testing.T) {
		pattern := "{year}/{month}/data.json"
		blobPath := "2024/03/data.json"

		parsedTime, err := parseTimeFromPattern(blobPath, pattern)
		require.NoError(t, err)
		require.NotNil(t, parsedTime)

		// Day defaults to 1 when not provided
		expected := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
		require.Equal(t, expected, *parsedTime)
	})

	t.Run("Placeholder pattern - missing year", func(t *testing.T) {
		pattern := "{month}/{day}/data.json"
		blobPath := "03/15/data.json"

		parsedTime, err := parseTimeFromPattern(blobPath, pattern)
		require.Error(t, err)
		require.Nil(t, parsedTime)
		require.Contains(t, err.Error(), "year is required")
	})

	t.Run("Placeholder pattern - path mismatch", func(t *testing.T) {
		pattern := "{year}/{month}/{day}/data.json"
		blobPath := "2024/03/data.json" // Missing day

		parsedTime, err := parseTimeFromPattern(blobPath, pattern)
		require.Error(t, err)
		require.Nil(t, parsedTime)
		require.Contains(t, err.Error(), "does not match pattern")
	})

	t.Run("Go time format - date and time", func(t *testing.T) {
		pattern := "2006/01/02/15/04"
		blobPath := "2024/03/15/14/30/data.json"

		parsedTime, err := parseTimeFromPattern(blobPath, pattern)
		require.NoError(t, err)
		require.NotNil(t, parsedTime)

		expected := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
		require.Equal(t, expected, *parsedTime)
	})

	t.Run("Go time format - date only", func(t *testing.T) {
		pattern := "2006/01/02"
		blobPath := "2024/03/15/data.json"

		parsedTime, err := parseTimeFromPattern(blobPath, pattern)
		require.NoError(t, err)
		require.NotNil(t, parsedTime)

		expected := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
		require.Equal(t, expected, *parsedTime)
	})

	t.Run("Go time format - with prefix", func(t *testing.T) {
		pattern := "logs/2006/01/02"
		blobPath := "logs/2024/03/15/file.json"

		parsedTime, err := parseTimeFromPattern(blobPath, pattern)
		require.NoError(t, err)
		require.NotNil(t, parsedTime)

		expected := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
		require.Equal(t, expected, *parsedTime)
	})

	t.Run("Go time format - fewer path components in blob", func(t *testing.T) {
		pattern := "2006/01/02/15"
		blobPath := "2024/03/15" // Missing hour component

		parsedTime, err := parseTimeFromPattern(blobPath, pattern)
		require.Error(t, err)
		require.Nil(t, parsedTime)
		require.Contains(t, err.Error(), "fewer components")
	})

	t.Run("Go time format - invalid time values", func(t *testing.T) {
		pattern := "2006/01/02"
		blobPath := "2024/13/45/data.json" // Invalid month and day

		parsedTime, err := parseTimeFromPattern(blobPath, pattern)
		require.Error(t, err)
		require.Nil(t, parsedTime)
	})

	t.Run("Placeholder pattern - special characters in path", func(t *testing.T) {
		pattern := "data.backup/{year}-{month}-{day}/file.json"
		blobPath := "data.backup/2024-03-15/file.json"

		parsedTime, err := parseTimeFromPattern(blobPath, pattern)
		require.NoError(t, err)
		require.NotNil(t, parsedTime)

		expected := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
		require.Equal(t, expected, *parsedTime)
	})
}
