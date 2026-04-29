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

// Package snapshotprocessor_v2 registers the v2 snapshotprocessor from
// bindplane-otel-contrib under the type name "snapshotprocessor_v2" so it
// can ship alongside the legacy v1 implementation in
// internal/processor/snapshotprocessor (registered as "snapshotprocessor").
//
// The wrapper delegates every method to the contrib factory; it does not
// reimplement any processor logic. To satisfy contrib's component-type
// validation (which requires set.ID.Type() to match the contrib factory's
// type "snapshotprocessor"), the wrapper rewrites set.ID before
// delegating. The framework-supplied logger is unaffected and continues
// to identify the component as "snapshotprocessor_v2/<name>". Internal
// metrics and the contrib-side processor cache key, however, will be
// recorded as "snapshotprocessor/<name>".
package snapshotprocessor_v2 // import "github.com/observiq/bindplane-otel-collector/internal/processor/snapshotprocessor_v2"

import (
	"context"

	contrib "github.com/observiq/bindplane-otel-contrib/processor/snapshotprocessor"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
)

// componentType is the type the wrapper registers under. It is distinct
// from the contrib factory's "snapshotprocessor" so both can be configured
// in the same collector.
var componentType = component.MustNewType("snapshotprocessor_v2")

// contribType is the type contrib's factory registers as; we rewrite
// set.ID to use it before delegating, since contrib's create methods
// validate set.ID.Type() against the factory's own type.
var contribType = component.MustNewType("snapshotprocessor")

// inner is the contrib factory the wrapper delegates to.
var inner = contrib.NewFactory()

// NewFactory returns a processor.Factory registered under the type
// "snapshotprocessor_v2" that delegates to bindplane-otel-contrib's
// snapshotprocessor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		componentType,
		inner.CreateDefaultConfig,
		processor.WithTraces(createTracesProcessor, inner.TracesStability()),
		processor.WithLogs(createLogsProcessor, inner.LogsStability()),
		processor.WithMetrics(createMetricsProcessor, inner.MetricsStability()),
	)
}

// rewriteID returns set with its ID's Type swapped to contrib's type. The
// instance Name is preserved so two pipelines using snapshotprocessor_v2
// remain distinguishable inside contrib.
func rewriteID(set processor.Settings) processor.Settings {
	set.ID = component.NewIDWithName(contribType, set.ID.Name())
	return set
}

func createTracesProcessor(ctx context.Context, set processor.Settings, cfg component.Config, next consumer.Traces) (processor.Traces, error) {
	return inner.CreateTraces(ctx, rewriteID(set), cfg, next)
}

func createLogsProcessor(ctx context.Context, set processor.Settings, cfg component.Config, next consumer.Logs) (processor.Logs, error) {
	return inner.CreateLogs(ctx, rewriteID(set), cfg, next)
}

func createMetricsProcessor(ctx context.Context, set processor.Settings, cfg component.Config, next consumer.Metrics) (processor.Metrics, error) {
	return inner.CreateMetrics(ctx, rewriteID(set), cfg, next)
}
