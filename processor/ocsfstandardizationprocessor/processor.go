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

package ocsfstandardizationprocessor

import (
	"context"
	"fmt"
	"strings"

	"github.com/observiq/bindplane-otel-collector/expr"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type compiledFieldMapping struct {
	from         *expr.Expression
	to           string
	defaultValue any
}

type compiledEventMapping struct {
	filter        *expr.Expression
	classID       int
	fieldMappings []compiledFieldMapping
}

type ocsfStandardizationProcessor struct {
	logger        *zap.Logger
	ocsfVersion   OCSFVersion
	eventMappings []compiledEventMapping
}

func newOCSFStandardizationProcessor(logger *zap.Logger, config *Config) (*ocsfStandardizationProcessor, error) {
	compiled := make([]compiledEventMapping, 0, len(config.EventMappings))
	for _, eventMapping := range config.EventMappings {
		fieldMappings := make([]compiledFieldMapping, 0, len(eventMapping.FieldMappings))
		for _, fieldMapping := range eventMapping.FieldMappings {
			cfm := compiledFieldMapping{
				to:           fieldMapping.To,
				defaultValue: fieldMapping.Default,
			}
			if fieldMapping.From != "" {
				from, err := expr.CreateValueExpression(fieldMapping.From)
				if err != nil {
					return nil, fmt.Errorf("compiling from expression: %w", err)
				}
				cfm.from = from
			}
			fieldMappings = append(fieldMappings, cfm)
		}

		compiledEventMap := compiledEventMapping{
			classID:       eventMapping.ClassID,
			fieldMappings: fieldMappings,
		}

		if eventMapping.Filter != "" {
			filter, err := expr.CreateBoolExpression(eventMapping.Filter)
			if err != nil {
				return nil, fmt.Errorf("compiling filter expression: %w", err)
			}
			compiledEventMap.filter = filter
		}

		compiled = append(compiled, compiledEventMap)
	}

	return &ocsfStandardizationProcessor{
		logger:        logger,
		ocsfVersion:   config.OCSFVersion,
		eventMappings: compiled,
	}, nil
}

func (osp *ocsfStandardizationProcessor) processLogs(_ context.Context, ld plog.Logs) (plog.Logs, error) {
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resource := ld.ResourceLogs().At(i)
		resourceAttrs := resource.Resource().Attributes().AsRaw()
		for j := 0; j < resource.ScopeLogs().Len(); j++ {
			scope := resource.ScopeLogs().At(j)
			scope.LogRecords().RemoveIf(func(log plog.LogRecord) bool {
				results := !osp.processLogRecord(log, resourceAttrs)
				if results {
					osp.logger.Debug("Dropping log record", zap.String("reason", "no match"))
				}
				return results
			})
		}
		resource.ScopeLogs().RemoveIf(func(scope plog.ScopeLogs) bool {
			records := scope.LogRecords().Len()
			if records == 0 {
				osp.logger.Debug("Dropping scope", zap.String("reason", "no records"))
			}
			return records == 0
		})
	}
	ld.ResourceLogs().RemoveIf(func(resource plog.ResourceLogs) bool {
		scopes := resource.ScopeLogs().Len()
		if scopes == 0 {
			osp.logger.Debug("Dropping resource", zap.String("reason", "no scopes"))
		}
		return scopes == 0
	})
	return ld, nil
}

// processLogRecord processes a single log record. Returns true to keep the record, false to drop it.
// This creates a new log body with the mapping applied.
func (osp *ocsfStandardizationProcessor) processLogRecord(log plog.LogRecord, resourceAttrs map[string]any) bool {
	record := expr.ConvertToRecord(log, resourceAttrs)

	for _, eventMapping := range osp.eventMappings {
		if eventMapping.filter != nil && !eventMapping.filter.MatchRecord(record) {
			continue
		}

		newBody := map[string]any{
			"class_uid": eventMapping.classID,
			"metadata": map[string]any{
				"version": string(osp.ocsfVersion),
			},
		}

		for _, fieldMapping := range eventMapping.fieldMappings {
			var value any
			if fieldMapping.from != nil {
				val, err := fieldMapping.from.Evaluate(record)
				if err != nil || val == nil {
					value = fieldMapping.defaultValue
				} else {
					value = val
				}
			} else {
				value = fieldMapping.defaultValue
			}

			if value == nil {
				continue
			}

			setNestedValue(newBody, fieldMapping.to, value)
		}

		if err := log.Body().SetEmptyMap().FromRaw(newBody); err != nil {
			osp.logger.Error("failed to set log body", zap.Error(err), zap.Int("class_id", eventMapping.classID))
			return false
		}

		return true
	}

	return false
}

// setNestedValue sets a value at a dot-separated path in a nested map,
// creating intermediate maps as needed.
func setNestedValue(body map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	for _, part := range parts[:len(parts)-1] {
		next, ok := body[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			body[part] = next
		}
		body = next
	}
	body[parts[len(parts)-1]] = value
}
