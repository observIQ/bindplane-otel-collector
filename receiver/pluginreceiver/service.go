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

package pluginreceiver

import (
	"context"

	"go.opentelemetry.io/collector/otelcol"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Service is an interface for running an internal open telemetry service
type Service interface {
	Run(ctx context.Context) error
	Shutdown()
	GetState() otelcol.State
}

// createService creates a default Service for running an open telemetry pipeline
func createService(factories otelcol.Factories, configProviderSettings otelcol.ConfigProviderSettings, logger *zap.Logger) (Service, error) {
	settings := otelcol.CollectorSettings{
		Factories:               func() (otelcol.Factories, error) { return factories, nil },
		DisableGracefulShutdown: true,
		ConfigProviderSettings:  configProviderSettings,
		LoggingOptions:          createServiceLoggerOpts(logger),
	}

	return otelcol.NewCollector(settings)
}

// createServiceLoggerOpts creates the default logger opts for a Service
func createServiceLoggerOpts(baseLogger *zap.Logger) []zap.Option {
	levelOpt := zap.IncreaseLevel(zap.ErrorLevel)
	coreOpt := zap.WrapCore(func(zapcore.Core) zapcore.Core {
		return baseLogger.Core()
	})
	return []zap.Option{coreOpt, levelOpt}
}

// createServiceFunc is a function used to create a service
type createServiceFunc func(factories otelcol.Factories, configProviderSettings otelcol.ConfigProviderSettings, logger *zap.Logger) (Service, error)
