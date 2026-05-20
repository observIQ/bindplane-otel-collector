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

package runtime

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/observiq/bindplane-otel-collector/internal/extension/opampconnectionextension/internal/collector"
	"github.com/observiq/bindplane-otel-collector/internal/extension/opampconnectionextension/internal/logging"
	"github.com/observiq/bindplane-otel-collector/internal/extension/opampconnectionextension/internal/service"
	"go.opentelemetry.io/collector/otelcol"
	"go.uber.org/zap"
)

// Options configures the managed-agent runtime entry point.
//
// Callers (main.go in the legacy or ocb-driven build) parse their own
// command-line flags / environment variables and hand them in here; Run
// owns logging setup, collector construction, managed/standalone dispatch,
// and the lifetime of the OpAMP client.
type Options struct {
	// Factories is the otelcol factory set used by the collector. The legacy
	// build passes factories.DefaultFactories(); the ocb-driven build passes
	// the ocb-generated components() result.
	Factories otelcol.Factories

	// Version is the agent version string passed through to the collector's
	// BuildInfo and reported via OpAMP.
	Version string

	// CollectorConfigPaths is the list of yaml paths the otelcol Resolver reads.
	// In managed mode, the first entry is the path Bindplane rewrites on each
	// remote-config push.
	CollectorConfigPaths []string

	// ManagerConfigPath is the path to manager.yaml. If it exists (or can be
	// bootstrapped from OPAMP_* env vars), the runtime starts in managed mode;
	// otherwise it starts in standalone mode.
	ManagerConfigPath string

	// LoggingConfigPath is the path to the YAML zap config. The default
	// (./logging.yaml) is auto-created on first run if missing.
	LoggingConfigPath string

	// FeatureGates is the list of otel collector feature gate identifiers to
	// enable at startup.
	FeatureGates []string
}

// Run boots the managed agent runtime with the supplied options. It returns
// when the collector exits; on a configuration or startup error it logs and
// terminates via log.Fatal / logger.Fatal (matching the legacy main.go
// semantics rather than introducing new error-return surface).
//
// The function:
//  1. Loads zap options from LoggingConfigPath.
//  2. Sets feature gates.
//  3. Constructs a collector around Factories + CollectorConfigPaths.
//  4. Bootstraps manager.yaml from env vars if absent (managed mode entry).
//  5. Launches the appropriate RunnableService (managed or standalone).
func Run(opts Options) {
	logOpts, err := loadLoggingOptions(opts.LoggingConfigPath)
	if err != nil {
		log.Fatalf("Failed to get log options: %v", err)
	}

	logger, err := zap.NewProduction(logOpts...)
	if err != nil {
		log.Fatalf("Failed to set up logger: %v", err)
	}

	if err := collector.SetFeatureFlags(opts.FeatureGates, logger); err != nil {
		logger.Fatal("Failed to set feature flags.", zap.Error(err))
	}

	col := collector.New(opts.CollectorConfigPaths, opts.Version, logOpts, opts.Factories)

	var runnableService service.RunnableService
	if err := bootstrapManagerConfig(&opts.ManagerConfigPath); err == nil {
		logger.Info("Starting In Managed Mode")

		collectorConfigPath := opts.CollectorConfigPaths[0]
		logger.Debug("Checking for existing rollback files")
		if err := checkForCollectorRollbackConfig(collectorConfigPath); err != nil {
			// Non-fatal — we still have *a* config to run with.
			logger.Error("Error occurred while checking for collector config rollbacks", zap.Error(err))
		}

		runnableService, err = service.NewManagedCollectorService(col, logger, opts.ManagerConfigPath, collectorConfigPath, opts.LoggingConfigPath)
		if err != nil {
			logger.Fatal("Failed to initiate managed mode", zap.Error(err))
		}
	} else if errors.Is(err, os.ErrNotExist) {
		logger.Info("Starting Standalone Mode")
		runnableService = service.NewStandaloneCollectorService(col)
	} else {
		logger.Fatal("Error while searching for management config", zap.Error(err))
	}

	if err := service.RunService(logger, runnableService); err != nil {
		logger.Fatal("RunService returned error", zap.Error(err))
	}
}

func loadLoggingOptions(loggingConfigPath string) ([]zap.Option, error) {
	if loggingConfigPath == "" {
		return nil, nil
	}
	l, err := logging.NewLoggerConfig(loggingConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger config: %w", err)
	}
	return l.Options()
}
