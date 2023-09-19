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

package azureblobexporter // import "github.com/observiq/bindplane-agent/exporter/azureblobexporter"

import (
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_baseMarshaler(t *testing.T) {
	m := newMarshaler(noCompression)

	require.IsType(t, &baseMarshaler{}, m)
	require.Equal(t, "json", m.Format())

	t.Run("Metrics Marshal", func(t *testing.T) {
		t.Parallel()
		md, expectedBytes := generateTestMetrics(t)
		actualBytes, err := m.MarshalMetrics(md)
		require.NoError(t, err)

		require.Equal(t, expectedBytes, actualBytes)
	})

	t.Run("Logs Marshal", func(t *testing.T) {
		t.Parallel()
		ld, expectedBytes := generateTestLogs(t)
		actualBytes, err := m.MarshalLogs(ld)
		require.NoError(t, err)

		require.Equal(t, expectedBytes, actualBytes)
	})

	t.Run("Trace Marshal", func(t *testing.T) {
		t.Parallel()
		td, expectedBytes := generateTestTraces(t)
		actualBytes, err := m.MarshalTraces(td)
		require.NoError(t, err)

		require.Equal(t, expectedBytes, actualBytes)
	})
}

func Test_gzipMarshaler(t *testing.T) {
	m := newMarshaler(gzipCompression)

	require.IsType(t, &gzipMarshaler{}, m)
	require.Equal(t, "json.gz", m.Format())

	t.Run("Metrics Marshal", func(t *testing.T) {
		t.Parallel()
		md, inputBytes := generateTestMetrics(t)
		actualBytes, err := m.MarshalMetrics(md)
		require.NoError(t, err)

		verifyGZipCompress(t, inputBytes, actualBytes)
	})

	t.Run("Logs Marshal", func(t *testing.T) {
		t.Parallel()
		ld, inputBytes := generateTestLogs(t)
		actualBytes, err := m.MarshalLogs(ld)
		require.NoError(t, err)

		verifyGZipCompress(t, inputBytes, actualBytes)
	})

	t.Run("Trace Marshal", func(t *testing.T) {
		t.Parallel()
		td, inputBytes := generateTestTraces(t)
		actualBytes, err := m.MarshalTraces(td)
		require.NoError(t, err)

		verifyGZipCompress(t, inputBytes, actualBytes)
	})
}

func verifyGZipCompress(t *testing.T, input, actual []byte) {
	t.Helper()
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	_, err := writer.Write(input)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	require.Equal(t, buf.Bytes(), actual)
}
