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

	"github.com/observiq/bindplane-agent/exporter/snowflakeexporter/internal/database"
	"github.com/observiq/bindplane-agent/exporter/snowflakeexporter/internal/database/mocks"
	"github.com/observiq/bindplane-agent/exporter/snowflakeexporter/internal/metrics"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/exporter/exportertest"
)

func TestNewMetricsExporter(t *testing.T) {
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
				Metrics: TelemetryConfig{
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
				Metrics: TelemetryConfig{
					Schema: "schema",
					Table:  "table",
				},
			},
			newDatabase: func(_, _, _ string) (database.Database, error) {
				return nil, fmt.Errorf("fail")
			},
			expectedErr: fmt.Errorf("failed to create new database connection for metrics: fail"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			exp, err := newMetricsExporter(
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

func TestMetricsCapabilities(t *testing.T) {
	e := &metricsExporter{}
	c := e.Capabilities()
	require.False(t, c.MutatesData)
}

func TestMetricsStart(t *testing.T) {
	c := &Config{
		AccountIdentifier: "id",
		Username:          "user",
		Password:          "pass",
		Database:          "db",
		Warehouse:         "wh",
		Metrics: TelemetryConfig{
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
				m.On("CreateSchema", ctx, c.Metrics.Schema).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateSumMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateGaugeMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateSummaryMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateHistogramMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateExponentialHistogramMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(nil)
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
			expectedErr: fmt.Errorf("failed to initialize database connection for metrics: fail"),
		},
		{
			desc: "fail CreateSchema",
			ctx:  context.Background(),
			mockGen: func(t *testing.T, ctx context.Context) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("InitDatabaseConn", ctx, c.Role).Return(nil)
				m.On("CreateSchema", ctx, c.Metrics.Schema).Return(fmt.Errorf("fail"))
				return m
			},
			expectedErr: fmt.Errorf("failed to create metrics schema: fail"),
		},
		{
			desc: "fail CreateTable sum",
			ctx:  context.Background(),
			mockGen: func(t *testing.T, ctx context.Context) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("InitDatabaseConn", ctx, c.Role).Return(nil)
				m.On("CreateSchema", ctx, c.Metrics.Schema).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateSumMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(fmt.Errorf("fail"))
				return m
			},
			expectedErr: fmt.Errorf("failed to create sum metrics table: fail"),
		},
		{
			desc: "fail CreateTable gauge",
			ctx:  context.Background(),
			mockGen: func(t *testing.T, ctx context.Context) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("InitDatabaseConn", ctx, c.Role).Return(nil)
				m.On("CreateSchema", ctx, c.Metrics.Schema).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateSumMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateGaugeMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(fmt.Errorf("fail"))
				return m
			},
			expectedErr: fmt.Errorf("failed to create gauge metrics table: fail"),
		},
		{
			desc: "fail CreateTable summary",
			ctx:  context.Background(),
			mockGen: func(t *testing.T, ctx context.Context) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("InitDatabaseConn", ctx, c.Role).Return(nil)
				m.On("CreateSchema", ctx, c.Metrics.Schema).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateSumMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateGaugeMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateSummaryMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(fmt.Errorf("fail"))
				return m
			},
			expectedErr: fmt.Errorf("failed to create summary metrics table: fail"),
		},
		{
			desc: "fail CreateTable histogram",
			ctx:  context.Background(),
			mockGen: func(t *testing.T, ctx context.Context) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("InitDatabaseConn", ctx, c.Role).Return(nil)
				m.On("CreateSchema", ctx, c.Metrics.Schema).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateSumMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateGaugeMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateSummaryMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateHistogramMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(fmt.Errorf("fail"))
				return m
			},
			expectedErr: fmt.Errorf("failed to create histogram metrics table: fail"),
		},
		{
			desc: "fail CreateTable exponential histogram",
			ctx:  context.Background(),
			mockGen: func(t *testing.T, ctx context.Context) *mocks.MockDatabase {
				m := mocks.NewMockDatabase(t)
				m.On("InitDatabaseConn", ctx, c.Role).Return(nil)
				m.On("CreateSchema", ctx, c.Metrics.Schema).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateSumMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateGaugeMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateSummaryMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateHistogramMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(nil)
				m.On("CreateTable", ctx, fmt.Sprintf(metrics.CreateExponentialHistogramMetricTableTemplate, c.Database, c.Metrics.Schema, c.Metrics.Table)).Return(fmt.Errorf("fail"))
				return m
			},
			expectedErr: fmt.Errorf("failed to create exponential histogram metrics table: fail"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			metricsExp, err := newMetricsExporter(
				tc.ctx,
				c,
				exportertest.NewNopCreateSettings(),
				func(_, _, _ string) (database.Database, error) { return nil, nil },
			)
			require.NoError(t, err)
			metricsExp.db = tc.mockGen(t, tc.ctx)

			err = metricsExp.start(tc.ctx, nil)
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}
		})
	}
}

func TestMetricsShutdown(t *testing.T) {
	// no db
	ctx := context.Background()
	e := &metricsExporter{}
	require.NoError(t, e.shutdown(ctx))

	// mock db errors
	mock := mocks.NewMockDatabase(t)
	mock.On("Close").Return(fmt.Errorf("fail")).Once()
	e.db = mock

	require.ErrorContains(t, e.shutdown(ctx), "fail")
}
