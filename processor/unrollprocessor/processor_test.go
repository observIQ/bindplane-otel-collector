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

package unrollprocessor

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/golden"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/plogtest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor/processortest"
)

func BenchmarkUnroll(b *testing.B) {
	unrollProcessor := &unrollProcessor{
		cfg: createDefaultConfig().(*Config),
	}
	testLogs := createTestResourceLogs()

	for n := 0; n < b.N; n++ {
		unrollProcessor.ProcessLogs(context.Background(), testLogs)
	}
}

func TestInvalidConfig(t *testing.T) {
	_, err := newUnrollProcessor(&Config{
		Field:     "invalid",
		Recursive: true,
	})
	require.Error(t, err)
}

func createTestResourceLogs() plog.Logs {
	rl := plog.NewLogs()
	for i := 0; i < 10; i++ {
		resourceLog := rl.ResourceLogs().AppendEmpty()
		for j := 0; j < 10; j++ {
			scopeLogs := resourceLog.ScopeLogs().AppendEmpty()
			scopeLogs.LogRecords().AppendEmpty().Body().SetEmptySlice().FromRaw([]any{1, 2, 3, 4, 5, 6, 7})
		}
	}
	return rl
}

func TestProcessor(t *testing.T) {
	for _, test := range []struct {
		name      string
		recursive bool
	}{
		{
			name: "nop",
		},
		{
			name: "simple",
		},
		{
			name: "mixed_slice_types",
		},
		{
			name: "some_not_slices",
		},
		{
			name: "recursive_false",
		},
		{
			name:      "recursive_true",
			recursive: true,
		},
		{
			name: "empty",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			input, err := golden.ReadLogs(filepath.Join("testdata", test.name, "input.yaml"))
			require.NoError(t, err)
			expected, err := golden.ReadLogs(filepath.Join("testdata", test.name, "expected.yaml"))
			require.NoError(t, err)

			f := NewFactory()
			cfg := f.CreateDefaultConfig().(*Config)
			cfg.Recursive = test.recursive
			set := processortest.NewNopSettings()
			sink := &consumertest.LogsSink{}
			p, err := f.CreateLogs(context.Background(), set, cfg, sink)
			require.NoError(t, err)

			err = p.ConsumeLogs(context.Background(), input)
			require.NoError(t, err)

			actual := sink.AllLogs()
			require.Equal(t, 1, len(actual))

			require.NoError(t, plogtest.CompareLogs(expected, actual[0]))
		})
	}
}
