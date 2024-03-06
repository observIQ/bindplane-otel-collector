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

package telemetrygeneratorreceiver

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestWindowsEventsGenerator(t *testing.T) {

	test := []struct {
		name         string
		cfg          GeneratorConfig
		expectedFile string
	}{
		// TODO tests
	}

	for _, tc := range test {
		t.Run(tc.name, func(t *testing.T) {
			g := newWindowsEventsGenerator(tc.cfg, zap.NewNop())
			logs := g.generateLogs()
			require.Equal(t, 0, logs.LogRecordCount())
		})
	}
}
