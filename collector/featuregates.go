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

// SetFeatureFlags sets hardcoded collector feature flags
func SetFeatureFlags(featureGates []string, logger *zap.Logger) error {
	// set hardcoded feature flags first to allow for user overrides
	if err := setHardcodedFeatureFlags(); err != nil {
		return fmt.Errorf("failed to set hardcoded feature flags: %w", err)
	}

	// set user feature flags
	if err := setUserFeatureFlags(featureGates); err != nil {
		return fmt.Errorf("failed to set user feature flags: %w", err)
	}

	if len(featureGates) > 0 {
		logger.Info("Feature gates successfully set", zap.String("featureGates", strings.Join(featureGates, ",")))
	}

	return nil
}

// setHardcodedFeatureFlags sets hardcoded feature flags
func setHardcodedFeatureFlags() error {
	if err := featuregate.GlobalRegistry().Set("filelog.allowFileDeletion", true); err != nil {
		return fmt.Errorf("failed to enable filelog.allowFileDeletion: %w", err)
	}
	if err := featuregate.GlobalRegistry().Set("filelog.allowHeaderMetadataParsing", true); err != nil {
		return fmt.Errorf("failed to enable filelog.allowHeaderMetadataParsing: %w", err)
	}
	if err := featuregate.GlobalRegistry().Set("filelog.mtimeSortType", true); err != nil {
		return fmt.Errorf("failed to enable filelog.mtimeSortType: %w", err)
	}
	if err := featuregate.GlobalRegistry().Set("exporter.prometheusremotewritexporter.enableSendingRW2", true); err != nil {
		return fmt.Errorf("failed to enable exporter.prometheusremotewritexporter.enableSendingRW2: %w", err)
	}

	return nil
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
