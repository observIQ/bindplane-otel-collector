// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package regexmatchprocessor_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor/processortest"

	"github.com/observiq/bindplane-otel-collector/processor/regexmatchprocessor"
	"github.com/observiq/bindplane-otel-collector/processor/regexmatchprocessor/internal/matcher"
)

func TestProcessor(t *testing.T) {
	f := regexmatchprocessor.NewFactory()
	require.Equal(t, component.MustNewType("regexmatch"), f.Type())

	cfg := f.CreateDefaultConfig().(*regexmatchprocessor.Config)
	cfg.Regexes = []matcher.NamedRegex{
		{
			Name:  "test",
			Regex: regexp.MustCompile("test"),
		},
	}
	cfg.DefaultValue = "default"

	ctx := context.Background()
	set := processortest.NewNopSettings(f.Type())
	sink := &consumertest.LogsSink{}
	proc, err := f.CreateLogs(ctx, set, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, proc)

	host := componenttest.NewNopHost()
	require.NoError(t, proc.Start(ctx, host))
	require.NoError(t, proc.Shutdown(ctx))
}

func TestProcessLogs(t *testing.T) {
	tests := []struct {
		name          string
		attributeName string
		regexes       []matcher.NamedRegex
		defaultValue  string
		entries       []struct {
			logBody       any
			expectedMatch string
		}
	}{
		{
			name:          "Basic matching",
			attributeName: "pattern",
			regexes: []matcher.NamedRegex{
				{
					Name:  "pattern_a",
					Regex: regexp.MustCompile("foo"),
				},
				{
					Name:  "pattern_b",
					Regex: regexp.MustCompile("bar"),
				},
			},
			defaultValue: "unknown",
			entries: []struct {
				logBody       any
				expectedMatch string
			}{
				{
					logBody:       "foo message",
					expectedMatch: "pattern_a",
				},
				{
					logBody:       "bar message",
					expectedMatch: "pattern_b",
				},
				{
					logBody:       "baz message",
					expectedMatch: "unknown",
				},
			},
		},
		{
			name:          "Empty default value",
			attributeName: "pattern",
			regexes: []matcher.NamedRegex{
				{
					Name:  "pattern_a",
					Regex: regexp.MustCompile("foo"),
				},
				{
					Name:  "pattern_b",
					Regex: regexp.MustCompile("bar"),
				},
			},
			defaultValue: "",
			entries: []struct {
				logBody       any
				expectedMatch string
			}{
				{
					logBody:       "foo message",
					expectedMatch: "pattern_a",
				},
				{
					logBody:       "no match",
					expectedMatch: "",
				},
			},
		},
		{
			name:          "Mixed body types",
			attributeName: "pattern",
			regexes: []matcher.NamedRegex{
				{
					Name:  "pattern_a",
					Regex: regexp.MustCompile("foo"),
				},
			},
			defaultValue: "unknown",
			entries: []struct {
				logBody       any
				expectedMatch string
			}{
				{
					logBody:       "foo message",
					expectedMatch: "pattern_a",
				},
				{
					logBody:       123,
					expectedMatch: "",
				},
				{
					logBody:       "bar message",
					expectedMatch: "unknown",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := regexmatchprocessor.NewFactory()
			cfg := f.CreateDefaultConfig().(*regexmatchprocessor.Config)
			cfg.AttributeName = tt.attributeName
			cfg.Regexes = tt.regexes
			cfg.DefaultValue = tt.defaultValue

			ctx := context.Background()
			set := processortest.NewNopSettings(f.Type())
			sink := &consumertest.LogsSink{}
			proc, err := f.CreateLogs(ctx, set, cfg, sink)
			require.NoError(t, err)

			logs := plog.NewLogs()
			resourceLog := logs.ResourceLogs().AppendEmpty()
			scopeLog := resourceLog.ScopeLogs().AppendEmpty()

			// Create multiple log records based on test entries
			logRecords := make([]plog.LogRecord, len(tt.entries))
			for i, entry := range tt.entries {
				logRecord := scopeLog.LogRecords().AppendEmpty()
				logRecords[i] = logRecord

				// Set the body according to the entry type
				switch v := entry.logBody.(type) {
				case string:
					logRecord.Body().SetStr(v)
				case int:
					logRecord.Body().SetInt(int64(v))
				default:
					t.Fatalf("test doesn't support log body type: %T", v)
				}
			}

			// Process the logs
			err = proc.ConsumeLogs(ctx, logs)
			require.NoError(t, err)

			// Verify each log record has the expected attribute
			for i, entry := range tt.entries {
				val, exists := logRecords[i].Attributes().Get(tt.attributeName)
				if entry.expectedMatch == "" {
					assert.False(t, exists)
				} else {
					assert.True(t, exists)
					assert.Equal(t, entry.expectedMatch, val.Str())
				}
			}
		})
	}
}
