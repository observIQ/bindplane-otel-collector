// Copyright  observIQ, Inc.
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

package severity

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/model/pdata"
)

func TestConvertSeverity(t *testing.T) {
	testCases := []struct {
		name   string
		input  int64
		output pdata.SeverityNumber
	}{
		{
			name:   "Default range",
			input:  5,
			output: pdata.SeverityNumberUNDEFINED,
		},
		{
			name:   "Catastrophe range",
			input:  106,
			output: pdata.SeverityNumberFATAL4,
		},
		{
			name:   "Notice range",
			input:  42,
			output: pdata.SeverityNumberWARN2,
		},
		{
			name:   "Debug range",
			input:  28,
			output: pdata.SeverityNumberDEBUG,
		},
		{
			name:   "Trace range",
			input:  14,
			output: pdata.SeverityNumberTRACE,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			out := ConvertSeverity(testCase.input)
			require.Equal(t, testCase.output, out)
		})
	}
}
