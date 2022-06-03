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

package resourceattributetransposerprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type logsProcessor struct {
	consumer consumer.Logs
	logger   *zap.Logger
	config   *Config
}

// newLogsProcessor returns a new logsResourceAttributeTransposerProcessor
func newLogsProcessor(logger *zap.Logger, consumer consumer.Logs, config *Config) *logsProcessor {
	return &logsProcessor{
		consumer: consumer,
		logger:   logger,
		config:   config,
	}
}

// Start starts the processor. It's a noop.
func (logsProcessor) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Capabilities returns the consumer's capabilities. Indicates that this processor mutates the incoming metrics.
func (logsProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (p logsProcessor) ConsumeLogs(ctx context.Context, md plog.Logs) error {
	resLogs := md.ResourceLogs()
	for i := 0; i < resLogs.Len(); i++ {
		resLog := resLogs.At(i)
		resourceAttrs := resLog.Resource().Attributes()
		for _, op := range p.config.Operations {
			resourceValue, ok := resourceAttrs.Get(op.From)
			if !ok {
				continue
			}

			scopeLogs := resLog.ScopeLogs()
			for j := 0; j < scopeLogs.Len(); j++ {
				scopeLog := scopeLogs.At(j)
				logs := scopeLog.LogRecords()
				for k := 0; k < logs.Len(); k++ {
					log := logs.At(k)
					log.Attributes().Insert(op.To, resourceValue)
				}
			}
		}
	}

	return p.consumer.ConsumeLogs(ctx, md)
}

// Shutdown stops the processor. It's a noop.
func (logsProcessor) Shutdown(_ context.Context) error {
	return nil
}
