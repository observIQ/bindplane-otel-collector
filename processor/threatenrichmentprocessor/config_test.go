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

package threatenrichmentprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validBloomConfig() *Config {
	return &Config{
		Filter: FilterConfig{
			Kind:              "bloom",
			EstimatedCount:    1000,
			FalsePositiveRate: 0.01,
		},
		Rules: []Rule{
			{
				Name:          "ips",
				IndicatorFile: "/tmp/indicators.txt",
				LookupFields:  []string{"body"},
			},
		},
	}
}

func TestConfig_Validate_ValidBloom(t *testing.T) {
	cfg := validBloomConfig()
	require.NoError(t, cfg.Validate())
	require.Equal(t, "bloom", cfg.Filter.Kind)
	require.Equal(t, "ips", cfg.Rules[0].Name)
}

func TestConfig_Validate_TrimsFields(t *testing.T) {
	cfg := validBloomConfig()
	cfg.Filter.Kind = "  bloom  "
	cfg.Rules[0].Name = "  ips "
	cfg.Rules[0].IndicatorFile = " /tmp/x "
	cfg.Rules[0].LookupFields = []string{" body ", "attr"}
	require.NoError(t, cfg.Validate())
	require.Equal(t, "bloom", cfg.Filter.Kind)
	require.Equal(t, "ips", cfg.Rules[0].Name)
	require.Equal(t, "/tmp/x", cfg.Rules[0].IndicatorFile)
	require.Equal(t, []string{"body", "attr"}, cfg.Rules[0].LookupFields)
}

func TestConfig_Validate_InvalidFilterKind(t *testing.T) {
	cfg := validBloomConfig()
	cfg.Filter.Kind = "nope"
	require.ErrorContains(t, cfg.Validate(), "filter kind")
}

func TestConfig_Validate_MissingFilterKind(t *testing.T) {
	cfg := validBloomConfig()
	cfg.Filter.Kind = ""
	require.ErrorContains(t, cfg.Validate(), "filter.kind is required")
}

func TestConfig_Validate_BloomMissingEstimatedCount(t *testing.T) {
	cfg := validBloomConfig()
	cfg.Filter.EstimatedCount = 0
	require.ErrorContains(t, cfg.Validate(), "estimated_count")
}

func TestConfig_Validate_BloomBadFPR(t *testing.T) {
	cfg := validBloomConfig()
	cfg.Filter.FalsePositiveRate = 0
	require.ErrorContains(t, cfg.Validate(), "false_positive_rate")

	cfg.Filter.FalsePositiveRate = 1
	require.ErrorContains(t, cfg.Validate(), "false_positive_rate")
}

func TestConfig_Validate_CuckooMissingCapacity(t *testing.T) {
	cfg := validBloomConfig()
	cfg.Filter = FilterConfig{Kind: "cuckoo", Capacity: 0}
	require.ErrorContains(t, cfg.Validate(), "capacity")
}

func TestConfig_Validate_VacuumMissingCapacity(t *testing.T) {
	cfg := validBloomConfig()
	cfg.Filter = FilterConfig{Kind: "vacuum", Capacity: 0}
	require.ErrorContains(t, cfg.Validate(), "capacity")
}

func TestConfig_Validate_CuckooOK(t *testing.T) {
	cfg := validBloomConfig()
	cfg.Filter = FilterConfig{Kind: "cuckoo", Capacity: 500}
	require.NoError(t, cfg.Validate())
}

func TestConfig_Validate_ScalableCuckooOK(t *testing.T) {
	cfg := validBloomConfig()
	cfg.Filter = FilterConfig{Kind: "scalable_cuckoo"}
	require.NoError(t, cfg.Validate())
}

func TestConfig_Validate_NoRules(t *testing.T) {
	cfg := validBloomConfig()
	cfg.Rules = nil
	require.ErrorContains(t, cfg.Validate(), "at least one rule")
}

func TestConfig_Validate_DuplicateRuleNames(t *testing.T) {
	cfg := validBloomConfig()
	cfg.Rules = append(cfg.Rules, Rule{
		Name:          "ips",
		IndicatorFile: "/tmp/b.txt",
		LookupFields:  []string{"body"},
	})
	require.ErrorContains(t, cfg.Validate(), "duplicate rule name")
}

func TestConfig_Validate_DuplicateRuleNamesAfterTrim(t *testing.T) {
	cfg := validBloomConfig()
	cfg.Rules = append(cfg.Rules, Rule{
		Name:          "  ips  ",
		IndicatorFile: "/tmp/b.txt",
		LookupFields:  []string{"body"},
	})
	require.ErrorContains(t, cfg.Validate(), "duplicate rule name")
}

func TestConfig_Validate_EmptyLookupField(t *testing.T) {
	cfg := validBloomConfig()
	cfg.Rules[0].LookupFields = []string{"body", "  "}
	require.ErrorContains(t, cfg.Validate(), "lookup_fields entry is empty")
}

func TestConfig_Validate_DuplicateLookupFields(t *testing.T) {
	cfg := validBloomConfig()
	cfg.Rules[0].LookupFields = []string{"body", "body"}
	require.ErrorContains(t, cfg.Validate(), "duplicate lookup_fields")
}

func TestConfig_Validate_RuleFilterInvalid(t *testing.T) {
	cfg := validBloomConfig()
	cfg.Rules[0].Filter = &FilterConfig{Kind: "bloom", EstimatedCount: 0}
	require.ErrorContains(t, cfg.Validate(), "estimated_count")
}

func TestConfig_Validate_RuleFilterVacuumNoCapacity(t *testing.T) {
	cfg := validBloomConfig()
	cfg.Rules[0].Filter = &FilterConfig{Kind: "vacuum", Capacity: 0}
	require.ErrorContains(t, cfg.Validate(), "filter.capacity required")
}
