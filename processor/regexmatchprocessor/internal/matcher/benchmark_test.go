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

package matcher_test

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/observiq/bindplane-otel-collector/processor/regexmatchprocessor/internal/benchtest"
	"github.com/observiq/bindplane-otel-collector/processor/regexmatchprocessor/internal/matcher"
)

func BenchmarkMatch(b *testing.B) {
	// Extract sample keys from the map
	sampleKeys := make([]string, 0, len(benchtest.Examples))
	for k := range benchtest.Examples {
		sampleKeys = append(sampleKeys, k)
	}

	// Shuffle the keys to randomize benchmark execution order
	rand.Shuffle(len(sampleKeys), func(i, j int) {
		sampleKeys[i], sampleKeys[j] = sampleKeys[j], sampleKeys[i]
	})

	// --- Naive Benchmark ---
	b.Run("naive", func(b *testing.B) {
		b.ResetTimer() // Start timing after setup
		for i := 0; i < b.N; i++ {
			for _, sample := range sampleKeys {
				expectedName := benchtest.Examples[sample]
				actualMatchName := "default"
				// Perform naive matching
				for _, r := range benchtest.Regexes {
					if r.Regex.MatchString(sample) {
						actualMatchName = r.Name
						break // First match wins
					}
				}
				// Inline validation (comment out for pure speed measurement)
				require.Equal(b, expectedName, actualMatchName, "Input: %s", sample)
			}
		}
		b.StopTimer()
	})

	// Now evaluate the optimized approach
	b.Run("optimized", func(b *testing.B) {
		// Setup matcher outside the timer
		matcher, err := matcher.New(benchtest.Regexes, "default")
		require.NoError(b, err)

		b.ResetTimer() // Start timing after setup
		for i := 0; i < b.N; i++ {
			for _, sample := range sampleKeys {
				expectedName := benchtest.Examples[sample]
				actualMatchName := matcher.Match(sample)

				// Inline validation (comment out for pure speed measurement)
				require.Equal(b, expectedName, actualMatchName, "Input: %s", sample)
			}
		}
		b.StopTimer()
	})
}
