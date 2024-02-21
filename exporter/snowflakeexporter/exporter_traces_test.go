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

package snowflakeexporter

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/observiq/bindplane-agent/exporter/snowflakeexporter/internal/database"
	"github.com/observiq/bindplane-agent/exporter/snowflakeexporter/internal/database/mocks"
	"github.com/observiq/bindplane-agent/exporter/snowflakeexporter/internal/utility"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestNewTracesExporter(t *testing.T) {
	testCases := []struct {
		desc        string
		ctx         context.Context
		c           *Config
		newDatabase func(_, _, _ string) (database.Database, error)
		expectedErr error
	}{
		{
			desc: "pass",
			ctx:  context.Background(),
			c: &Config{
				AccountIdentifier: "id",
				Username:          "user",
				Password:          "pass",
				Database:          "db",
				Traces: TelemetryConfig{
					Schema: "schema",
					Table:  "table",
				},
			},
			newDatabase: func(_, _, _ string) (database.Database, error) {
				return mocks.NewMockDatabase(t), nil
			},
		},
		{
			desc: "fail newDatabase",
			ctx:  context.Background(),
			c: &Config{
				AccountIdentifier: "id",
				Username:          "user",
				Password:          "pass",
				Database:          "db",
				Traces: TelemetryConfig{
					Schema: "schema",
					Table:  "table",
				},
			},
			newDatabase: func(_, _, _ string) (database.Database, error) {
				return nil, fmt.Errorf("fail")
			},
			expectedErr: fmt.Errorf("failed to create new database connection for traces: fail"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			exp, err := newTracesExporter(
				tc.ctx,
				tc.c,
				exportertest.NewNopCreateSettings(),
				tc.newDatabase,
			)

			if tc.expectedErr == nil {
				require.NoError(t, err)
				require.NotNil(t, exp)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
				require.Nil(t, exp)
			}
		})
	}
}

func TestTracesCapabilities(t *testing.T) {
	e := &tracesExporter{}
	c := e.Capabilities()
	require.False(t, c.MutatesData)
}

func TestTracesStart(t *testing.T) {
	c := &Config{
		AccountIdentifier: "id",
		Username:          "user",
		Password:          "pass",
		Database:          "db",
		Warehouse:         "wh",
		Traces: TelemetryConfig{
			Schema: "schema",
			Table:  "table",
		},
	}

	testCases := []struct {
		desc        string
		ctx         context.Context
		mockGen     func(t *testing.T, ctx context.Context) *mocks.MockDatabase
		expectedErr error
	}{
		{
			desc: "pass",
			ctx:  context.Background(),
			mockGen: func(t *testing.T, ctx context.Context) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("InitDatabaseConn", ctx, c.Role).Return(nil)
				m.On("CreateSchema", ctx, c.Traces.Schema).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(createTracesTableSnowflakeTemplate, c.Database, c.Traces.Schema, c.Traces.Table)).Return(nil)
				return m
			},
		},
		{
			desc: "fail InitDatabaseConn",
			ctx:  context.Background(),
			mockGen: func(t *testing.T, ctx context.Context) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("InitDatabaseConn", ctx, c.Role).Return(fmt.Errorf("fail"))
				return m
			},
			expectedErr: fmt.Errorf("failed to initialize database connection for traces: fail"),
		},
		{
			desc: "fail CreateSchema",
			ctx:  context.Background(),
			mockGen: func(t *testing.T, ctx context.Context) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("InitDatabaseConn", ctx, c.Role).Return(nil)
				m.On("CreateSchema", ctx, c.Traces.Schema).Return(fmt.Errorf("fail"))
				return m
			},
			expectedErr: fmt.Errorf("failed to create traces schema: fail"),
		},
		{
			desc: "fail CreateTable",
			ctx:  context.Background(),
			mockGen: func(t *testing.T, ctx context.Context) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("InitDatabaseConn", ctx, c.Role).Return(nil)
				m.On("CreateSchema", ctx, c.Traces.Schema).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(createTracesTableSnowflakeTemplate, c.Database, c.Traces.Schema, c.Traces.Table)).Return(fmt.Errorf("fail"))
				return m
			},
			expectedErr: fmt.Errorf("failed to create traces table: fail"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tracesExp, err := newTracesExporter(
				tc.ctx,
				c,
				exportertest.NewNopCreateSettings(),
				func(_, _, _ string) (database.Database, error) { return nil, nil },
			)
			require.NoError(t, err)
			tracesExp.db = tc.mockGen(t, tc.ctx)

			err = tracesExp.start(tc.ctx, nil)
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}
		})
	}
}

func TestTracesShutdown(t *testing.T) {
	// no db
	ctx := context.Background()
	e := &tracesExporter{}
	require.NoError(t, e.shutdown(ctx))

	// db & error
	mock := mocks.NewMockDatabase(t)
	mock.On("Close").Return(fmt.Errorf("fail")).Once()
	e.db = mock

	require.ErrorContains(t, e.shutdown(ctx), "fail")
}

func TestTracesDataPusher(t *testing.T) {
	c := &Config{
		AccountIdentifier: "id",
		Username:          "user",
		Password:          "pass",
		Database:          "db",
		Traces: TelemetryConfig{
			Schema: "schema",
			Table:  "table",
		},
	}

	testCases := []struct {
		desc        string
		ctx         context.Context
		traceGen    func() ptrace.Traces
		mockGen     func(t *testing.T, ctx context.Context, sql string) *mocks.MockDatabase
		expectedErr error
	}{
		{
			desc:     "pass",
			ctx:      context.Background(),
			traceGen: generateTraces1,
			mockGen: func(t *testing.T, ctx context.Context, sql string) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("BatchInsert", ctx, expectedTraceMaps1(), sql).Return(nil)
				return m
			},
		},
		{
			desc:     "fail BatchInsert",
			ctx:      context.Background(),
			traceGen: generateTraces1,
			mockGen: func(t *testing.T, ctx context.Context, sql string) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("BatchInsert", ctx, expectedTraceMaps1(), sql).Return(fmt.Errorf("fail"))
				return m
			},
			expectedErr: fmt.Errorf("failed to insert trace data: fail"),
		},
		{
			desc:     "pass w/ links & events",
			ctx:      context.Background(),
			traceGen: generateTraces2,
			mockGen: func(t *testing.T, ctx context.Context, sql string) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("BatchInsert", ctx, expectedTraceMaps2(), sql).Return(nil)
				return m
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tracesExp, err := newTracesExporter(
				tc.ctx,
				c,
				exportertest.NewNopCreateSettings(),
				func(_, _, _ string) (database.Database, error) { return nil, nil },
			)
			require.NoError(t, err)
			tracesExp.db = tc.mockGen(t, tc.ctx, tracesExp.insertSQL)

			err = tracesExp.tracesDataPusher(tc.ctx, tc.traceGen())
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}
		})
	}
}

func generateTraces1() ptrace.Traces {
	traces := ptrace.NewTraces()
	rSpan := traces.ResourceSpans().AppendEmpty()
	rSpan.SetSchemaUrl("resource_schema_url")
	sSpan := rSpan.ScopeSpans().AppendEmpty()
	sSpan.SetSchemaUrl("scope_schema_url")
	for i := 0; i < 3; i++ {
		s := sSpan.Spans().AppendEmpty()
		s.SetName(fmt.Sprintf("span_%d", i))
	}
	return traces
}

func generateTraces2() ptrace.Traces {
	traces := ptrace.NewTraces()
	rSpan := traces.ResourceSpans().AppendEmpty()
	rSpan.SetSchemaUrl("resource_schema_url")
	sSpan := rSpan.ScopeSpans().AppendEmpty()
	sSpan.SetSchemaUrl("scope_schema_url")
	for i := 0; i < 3; i++ {
		s := sSpan.Spans().AppendEmpty()
		s.SetName(fmt.Sprintf("span_%d", i))

		e := s.Events().AppendEmpty()
		e.Attributes().FromRaw(map[string]any{"event_key": "event_value"})

		l := s.Links().AppendEmpty()
		l.Attributes().FromRaw(map[string]any{"link_key": "link_value"})
	}
	return traces
}

func expectedTraceMaps1() []map[string]any {
	return []map[string]any{
		{
			"rSchema":           "resource_schema_url",
			"rDroppedCount":     uint32(0),
			"rAttributes":       "{}",
			"sSchema":           "scope_schema_url",
			"sName":             "",
			"sVersion":          "",
			"sDroppedCount":     uint32(0),
			"sAttributes":       "{}",
			"traceID":           "",
			"spanID":            "",
			"traceState":        "",
			"parentSpanID":      "",
			"name":              "span_0",
			"kind":              "Unspecified",
			"startTime":         time.Unix(0, int64(0)).UTC(),
			"endTime":           time.Unix(0, int64(0)).UTC(),
			"droppedCount":      uint32(0),
			"attributes":        "{}",
			"statusMessage":     "",
			"statusCode":        "Unset",
			"eventTimes":        utility.Array{},
			"eventNames":        utility.Array{},
			"eventDroppedCount": utility.Array{},
			"eventAttributes":   utility.Array{},
			"linkTraceIDs":      utility.Array{},
			"linkSpanIDs":       utility.Array{},
			"linkTraceStates":   utility.Array{},
			"linkDroppedCount":  utility.Array{},
			"linkAttributes":    utility.Array{},
		},
		{
			"rSchema":           "resource_schema_url",
			"rDroppedCount":     uint32(0),
			"rAttributes":       "{}",
			"sSchema":           "scope_schema_url",
			"sName":             "",
			"sVersion":          "",
			"sDroppedCount":     uint32(0),
			"sAttributes":       "{}",
			"traceID":           "",
			"spanID":            "",
			"traceState":        "",
			"parentSpanID":      "",
			"name":              "span_1",
			"kind":              "Unspecified",
			"startTime":         time.Unix(0, int64(0)).UTC(),
			"endTime":           time.Unix(0, int64(0)).UTC(),
			"droppedCount":      uint32(0),
			"attributes":        "{}",
			"statusMessage":     "",
			"statusCode":        "Unset",
			"eventTimes":        utility.Array{},
			"eventNames":        utility.Array{},
			"eventDroppedCount": utility.Array{},
			"eventAttributes":   utility.Array{},
			"linkTraceIDs":      utility.Array{},
			"linkSpanIDs":       utility.Array{},
			"linkTraceStates":   utility.Array{},
			"linkDroppedCount":  utility.Array{},
			"linkAttributes":    utility.Array{},
		},
		{
			"rSchema":           "resource_schema_url",
			"rDroppedCount":     uint32(0),
			"rAttributes":       "{}",
			"sSchema":           "scope_schema_url",
			"sName":             "",
			"sVersion":          "",
			"sDroppedCount":     uint32(0),
			"sAttributes":       "{}",
			"traceID":           "",
			"spanID":            "",
			"traceState":        "",
			"parentSpanID":      "",
			"name":              "span_2",
			"kind":              "Unspecified",
			"startTime":         time.Unix(0, int64(0)).UTC(),
			"endTime":           time.Unix(0, int64(0)).UTC(),
			"droppedCount":      uint32(0),
			"attributes":        "{}",
			"statusMessage":     "",
			"statusCode":        "Unset",
			"eventTimes":        utility.Array{},
			"eventNames":        utility.Array{},
			"eventDroppedCount": utility.Array{},
			"eventAttributes":   utility.Array{},
			"linkTraceIDs":      utility.Array{},
			"linkSpanIDs":       utility.Array{},
			"linkTraceStates":   utility.Array{},
			"linkDroppedCount":  utility.Array{},
			"linkAttributes":    utility.Array{},
		},
	}
}

func expectedTraceMaps2() []map[string]any {
	return []map[string]any{
		{
			"rSchema":           "resource_schema_url",
			"rDroppedCount":     uint32(0),
			"rAttributes":       "{}",
			"sSchema":           "scope_schema_url",
			"sName":             "",
			"sVersion":          "",
			"sDroppedCount":     uint32(0),
			"sAttributes":       "{}",
			"traceID":           "",
			"spanID":            "",
			"traceState":        "",
			"parentSpanID":      "",
			"name":              "span_0",
			"kind":              "Unspecified",
			"startTime":         time.Unix(0, int64(0)).UTC(),
			"endTime":           time.Unix(0, int64(0)).UTC(),
			"droppedCount":      uint32(0),
			"attributes":        "{}",
			"statusMessage":     "",
			"statusCode":        "Unset",
			"eventTimes":        utility.Array{time.Unix(0, int64(0)).UTC()},
			"eventNames":        utility.Array{""},
			"eventDroppedCount": utility.Array{uint32(0)},
			"eventAttributes":   utility.Array{map[string]any{"event_key": "event_value"}},
			"linkTraceIDs":      utility.Array{""},
			"linkSpanIDs":       utility.Array{""},
			"linkTraceStates":   utility.Array{""},
			"linkDroppedCount":  utility.Array{uint32(0)},
			"linkAttributes":    utility.Array{map[string]any{"link_key": "link_value"}},
		},
		{
			"rSchema":           "resource_schema_url",
			"rDroppedCount":     uint32(0),
			"rAttributes":       "{}",
			"sSchema":           "scope_schema_url",
			"sName":             "",
			"sVersion":          "",
			"sDroppedCount":     uint32(0),
			"sAttributes":       "{}",
			"traceID":           "",
			"spanID":            "",
			"traceState":        "",
			"parentSpanID":      "",
			"name":              "span_1",
			"kind":              "Unspecified",
			"startTime":         time.Unix(0, int64(0)).UTC(),
			"endTime":           time.Unix(0, int64(0)).UTC(),
			"droppedCount":      uint32(0),
			"attributes":        "{}",
			"statusMessage":     "",
			"statusCode":        "Unset",
			"eventTimes":        utility.Array{time.Unix(0, int64(0)).UTC()},
			"eventNames":        utility.Array{""},
			"eventDroppedCount": utility.Array{uint32(0)},
			"eventAttributes":   utility.Array{map[string]any{"event_key": "event_value"}},
			"linkTraceIDs":      utility.Array{""},
			"linkSpanIDs":       utility.Array{""},
			"linkTraceStates":   utility.Array{""},
			"linkDroppedCount":  utility.Array{uint32(0)},
			"linkAttributes":    utility.Array{map[string]any{"link_key": "link_value"}},
		},
		{
			"rSchema":           "resource_schema_url",
			"rDroppedCount":     uint32(0),
			"rAttributes":       "{}",
			"sSchema":           "scope_schema_url",
			"sName":             "",
			"sVersion":          "",
			"sDroppedCount":     uint32(0),
			"sAttributes":       "{}",
			"traceID":           "",
			"spanID":            "",
			"traceState":        "",
			"parentSpanID":      "",
			"name":              "span_2",
			"kind":              "Unspecified",
			"startTime":         time.Unix(0, int64(0)).UTC(),
			"endTime":           time.Unix(0, int64(0)).UTC(),
			"droppedCount":      uint32(0),
			"attributes":        "{}",
			"statusMessage":     "",
			"statusCode":        "Unset",
			"eventTimes":        utility.Array{time.Unix(0, int64(0)).UTC()},
			"eventNames":        utility.Array{""},
			"eventDroppedCount": utility.Array{uint32(0)},
			"eventAttributes":   utility.Array{map[string]any{"event_key": "event_value"}},
			"linkTraceIDs":      utility.Array{""},
			"linkSpanIDs":       utility.Array{""},
			"linkTraceStates":   utility.Array{""},
			"linkDroppedCount":  utility.Array{uint32(0)},
			"linkAttributes":    utility.Array{map[string]any{"link_key": "link_value"}},
		},
	}
}
