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
	"filelog.mtimeSortType",
	"connector.spanmetrics.includeCollectorInstanceID",
}

// SetFeatureFlags sets hardcoded collector feature flags
func SetFeatureFlags(featureGates []string, logger *zap.Logger) error {
	// set hardcoded feature flags first to allow for user overrides
	setHardcodedFeatureFlags(logger)

	// set user feature flags
	if err := setUserFeatureFlags(featureGates); err != nil {
		return fmt.Errorf("failed to set user feature flags: %w", err)
	}

	if len(featureGates) > 0 {
		logger.Info("Feature gates successfully set", zap.String("featureGates", strings.Join(featureGates, ",")))
	}

	return nil
}

// setHardcodedFeatureFlags enables every gate in hardcodedFeatureGates that
// the current OTel registry knows about. Unknown gates are logged at debug
// and skipped.
func setHardcodedFeatureFlags(logger *zap.Logger) {
	for _, gate := range hardcodedFeatureGates {
		if err := featuregate.GlobalRegistry().Set(gate, true); err != nil {
			logger.Debug("skipping unknown hardcoded feature gate", zap.String("gate", gate), zap.Error(err))
		}
	}
}

// setUserFeatureFlags sets user feature flags
// checks for - and + prefixes to indicate negation and enablement of the feature gate
func setUserFeatureFlags(featureGates []string) error {
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
			return fmt.Errorf("failed to set feature gate %s: %w", fg, err)
		}
	}
	return nil
}
