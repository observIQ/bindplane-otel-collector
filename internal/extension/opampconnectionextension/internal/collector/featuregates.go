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
	"errors"
	"fmt"
	"slices"
	"strings"

	"go.opentelemetry.io/collector/featuregate"
	"go.uber.org/zap"
)

// hardcodedFeatureGates is the set of upstream OTel feature gates we enable by
// default before user-supplied gates are applied. A gate the registry doesn't
// know fails SetFeatureFlags, except the ones in skippedFeatureGates, which
// belong to components a platform build intentionally leaves out.
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
	if err := setFeatureFlags(hardcodedFeatureGates, logger); err != nil {
		return fmt.Errorf("set hardcoded feature flags: %w", err)
	}

	// set user feature flags
	if err := setFeatureFlags(featureGates, logger); err != nil {
		return fmt.Errorf("set user feature flags: %w", err)
	}

	if len(featureGates) > 0 {
		logger.Info("Feature gates successfully set", zap.String("featureGates", strings.Join(featureGates, ",")))
	}

	return nil
}

// setFeatureFlags sets user feature flags
// checks for - and + prefixes to indicate negation and enablement of the feature gate
func setFeatureFlags(featureGates []string, logger *zap.Logger) error {
	for _, fg := range featureGates {
		val := true
		switch fg[0] {
		case '-':
			fg = fg[1:]
			val = false
		case '+':
			fg = fg[1:]
		}

		if slices.Contains(skippedFeatureGates, fg) && !isRegistered(fg) {
			logger.Info("Skipping feature gate this build doesn't include", zap.String("featureGate", fg))
			continue
		}

		if err := featuregate.GlobalRegistry().Set(fg, val); err != nil {
			return fmt.Errorf("set feature gate %s: %w", fg, err)
		}
	}
	return nil
}

// isRegistered reports whether the global registry knows the gate.
func isRegistered(id string) bool {
	found := false
	featuregate.GlobalRegistry().VisitAll(func(g *featuregate.Gate) {
		found = found || g.ID() == id
	})
	return found
}

// CheckFeatureGates reports a hardcoded gate that is neither registered nor in
// skipped, and a skipped gate that is registered. Run it against a platform
// build to keep that platform's skip list in step with its components.
func CheckFeatureGates(skipped []string) error {
	var errs []error
	for _, fg := range hardcodedFeatureGates {
		fg = strings.TrimLeft(fg, "+-")
		if !isRegistered(fg) && !slices.Contains(skipped, fg) {
			errs = append(errs, fmt.Errorf("hardcoded feature gate %s isn't registered and isn't skipped", fg))
		}
	}
	for _, fg := range skipped {
		if isRegistered(fg) {
			errs = append(errs, fmt.Errorf("skipped feature gate %s is registered, so remove it from the skip list", fg))
		}
	}
	return errors.Join(errs...)
}
