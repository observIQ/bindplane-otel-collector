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
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/pdata/plog"
)

func TestNewLogsExporter(t *testing.T) {
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
				Logs: TelemetryConfig{
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
				Logs: TelemetryConfig{
					Schema: "schema",
					Table:  "table",
				},
			},
			newDatabase: func(_, _, _ string) (database.Database, error) {
				return nil, fmt.Errorf("fail")
			},
			expectedErr: fmt.Errorf("failed to create new database connection for logs: fail"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			exp, err := newLogsExporter(
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

func TestLogsCapabilities(t *testing.T) {
	e := &logsExporter{}
	c := e.Capabilities()
	require.False(t, c.MutatesData)
}

func TestLogsStart(t *testing.T) {
	c := &Config{
		AccountIdentifier: "id",
		Username:          "user",
		Password:          "pass",
		Database:          "db",
		Warehouse:         "wh",
		Logs: TelemetryConfig{
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
				m.On("CreateSchema", ctx, c.Logs.Schema).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(createLogsTableSnowflakeTemplate, c.Database, c.Logs.Schema, c.Logs.Table)).Return(nil)
				return m
			},
		},
		{
			desc:        "fail InitDatabaseConn",
			ctx:         context.Background(),
			expectedErr: fmt.Errorf("failed to initialize database connection for logs: fail"),
			mockGen: func(t *testing.T, ctx context.Context) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("InitDatabaseConn", ctx, c.Role).Return(fmt.Errorf("fail"))
				return m
			},
		},
		{
			desc:        "fail CreateSchema",
			ctx:         context.Background(),
			expectedErr: fmt.Errorf("failed to create logs schema: fail"),
			mockGen: func(t *testing.T, ctx context.Context) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("InitDatabaseConn", ctx, c.Role).Return(nil)
				m.On("CreateSchema", ctx, c.Logs.Schema).Return(fmt.Errorf("fail"))
				return m
			},
		},
		{
			desc:        "fail CreateTable",
			ctx:         context.Background(),
			expectedErr: fmt.Errorf("failed to create logs table: fail"),
			mockGen: func(t *testing.T, ctx context.Context) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("InitDatabaseConn", ctx, c.Role).Return(nil)
				m.On("CreateSchema", ctx, c.Logs.Schema).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(createLogsTableSnowflakeTemplate, c.Database, c.Logs.Schema, c.Logs.Table)).Return(fmt.Errorf("fail"))
				return m
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			logsExp, err := newLogsExporter(
				tc.ctx,
				c,
				exportertest.NewNopCreateSettings(),
				func(_, _, _ string) (database.Database, error) { return nil, nil },
			)
			require.NoError(t, err)
			logsExp.db = tc.mockGen(t, tc.ctx)

			err = logsExp.start(tc.ctx, nil)
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}
		})
	}
}

func TestLogsShutdown(t *testing.T) {
	// no db
	ctx := context.Background()
	e := &logsExporter{}
	require.NoError(t, e.shutdown(ctx))

	// db & error
	mock := mocks.NewMockDatabase(t)
	mock.On("Close").Return(fmt.Errorf("fail")).Once()
	e.db = mock

	require.ErrorContains(t, e.shutdown(ctx), "fail")
}

func TestLogsDataPusher(t *testing.T) {
	c := &Config{
		AccountIdentifier: "id",
		Username:          "user",
		Password:          "pass",
		Warehouse:         "wh",
		Database:          "db",
		Logs: TelemetryConfig{
			Schema: "schema",
			Table:  "table",
		},
	}

	testCases := []struct {
		desc        string
		ctx         context.Context
		logGen      func() plog.Logs
		mockGen     func(t *testing.T, ctx context.Context, sql string) *mocks.MockDatabase
		expectedErr error
	}{
		{
			desc:   "pass",
			ctx:    context.Background(),
			logGen: generateLogs,
			mockGen: func(t *testing.T, ctx context.Context, sql string) *mocks.MockDatabase {
				mock := mocks.NewMockDatabase(t)
				mock.On("BatchInsert", ctx, expectedLogMaps(), sql).Return(nil)
				return mock
			},
		},
		{
			desc:   "fail BatchInsert",
			ctx:    context.Background(),
			logGen: generateLogs,
			mockGen: func(t *testing.T, ctx context.Context, sql string) *mocks.MockDatabase {
				mock := mocks.NewMockDatabase(t)
				mock.On("BatchInsert", ctx, expectedLogMaps(), sql).Return(fmt.Errorf("fail"))
				return mock
			},
			expectedErr: fmt.Errorf("failed to insert log data: fail"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			logsExp, err := newLogsExporter(
				tc.ctx,
				c,
				exportertest.NewNopCreateSettings(),
				func(_, _, _ string) (database.Database, error) { return nil, nil },
			)
			require.NoError(t, err)
			logsExp.db = tc.mockGen(t, tc.ctx, logsExp.insertSQL)

			err = logsExp.logsDataPusher(tc.ctx, tc.logGen())
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}
		})
	}
}

func generateLogs() plog.Logs {
	logs := plog.NewLogs()
	rLogs := logs.ResourceLogs().AppendEmpty()
	rLogs.SetSchemaUrl("resource_schema_url")
	sLogs := rLogs.ScopeLogs().AppendEmpty()
	sLogs.SetSchemaUrl("scope_schema_url")
	for i := 0; i < 3; i++ {
		lr := sLogs.LogRecords().AppendEmpty()
		lr.Body().SetStr(fmt.Sprintf("log_body_%d", i))
	}
	return logs
}

func expectedLogMaps() []map[string]any {
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
			"timestamp":         time.Unix(0, int64(0)).UTC(),
			"observedTimestamp": time.Unix(0, int64(0)).UTC(),
			"severityNumber":    "Unspecified",
			"severityText":      "",
			"body":              "log_body_0",
			"attributes":        "{}",
			"droppedCount":      uint32(0),
			"flags":             plog.LogRecordFlags(0),
			"traceID":           "",
			"spanID":            "",
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
			"timestamp":         time.Unix(0, int64(0)).UTC(),
			"observedTimestamp": time.Unix(0, int64(0)).UTC(),
			"severityNumber":    "Unspecified",
			"severityText":      "",
			"body":              "log_body_1",
			"attributes":        "{}",
			"droppedCount":      uint32(0),
			"flags":             plog.LogRecordFlags(0),
			"traceID":           "",
			"spanID":            "",
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
			"timestamp":         time.Unix(0, int64(0)).UTC(),
			"observedTimestamp": time.Unix(0, int64(0)).UTC(),
			"severityNumber":    "Unspecified",
			"severityText":      "",
			"body":              "log_body_2",
			"attributes":        "{}",
			"droppedCount":      uint32(0),
			"flags":             plog.LogRecordFlags(0),
			"traceID":           "",
			"spanID":            "",
		},
	}
}
