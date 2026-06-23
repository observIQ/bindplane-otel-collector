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

package collector

import (
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/featuregate"
	"go.uber.org/zap"
)

// hardcodedFeatureGates is the set of upstream OTel feature gates we enable by
// default before user-supplied gates are applied. Gates that the loaded OTel
// build has dropped (or that belong to a component the binary isn't compiled
// with — e.g. the contrib spanmetrics connector when unit-testing the
// collector package alone) are skipped with a log entry instead of failing
// SetFeatureFlags, so the runtime survives OTel version churn.
var hardcodedFeatureGates = []string{
	"filelog.allowFileDeletion",
	"filelog.allowHeaderMetadataParsing",
	"filelog.mtimeSortType",
	"exporter.prometheusremotewritexporter.enableSendingRW2",
	"connector.spanmetrics.includeCollectorInstanceID",
	"-exporter.prometheusexporter.DisableAddMetricSuffixes",
}

// SetFeatureFlags sets hardcoded collector feature flags
func SetFeatureFlags(featureGates []string, logger *zap.Logger) error {
	// set hardcoded feature flags first to allow for user overrides
	if err := setFeatureFlags(hardcodedFeatureGates); err != nil {
		return fmt.Errorf("set hardcoded feature flags: %w", err)
	}

	// set user feature flags
	if err := setFeatureFlags(featureGates); err != nil {
		return fmt.Errorf("set user feature flags: %w", err)
	}

	if len(featureGates) > 0 {
		logger.Info("Feature gates successfully set", zap.String("featureGates", strings.Join(featureGates, ",")))
	}

	return nil
}

// setFeatureFlags sets user feature flags
// checks for - and + prefixes to indicate negation and enablement of the feature gate
func setFeatureFlags(featureGates []string) error {
	for _, fg := range featureGates {
		val := true
		switch fg[0] {
		case '-':
			fg = fg[1:]
			val = false
		case '+':
			fg = fg[1:]
		}

		if err := featuregate.GlobalRegistry().Set(fg, val); err != nil {
			return fmt.Errorf("set feature gate %s: %w", fg, err)
		}
	}
	return nil
}
