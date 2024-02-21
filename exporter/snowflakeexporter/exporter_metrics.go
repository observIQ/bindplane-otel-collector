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
	"errors"
	"fmt"
	"sync"

	"github.com/observiq/bindplane-agent/exporter/snowflakeexporter/internal/database"
	"github.com/observiq/bindplane-agent/exporter/snowflakeexporter/internal/metrics"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type metricModel interface {
	AddMetric(r pmetric.ResourceMetrics, s pmetric.ScopeMetrics, m pmetric.Metric)
	BatchInsert(ctx context.Context, db database.Database) error
}

type metricsExporter struct {
	cfg    *Config
	logger *zap.Logger
	db     database.Database
}

func newMetricsExporter(
	_ context.Context,
	cfg *Config,
	params exporter.CreateSettings,
	newDatabase func(dsn, wh, db string) (database.Database, error),
) (*metricsExporter, error) {
	db, err := newDatabase(cfg.dsn, cfg.Warehouse, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to create new database connection for metrics: %w", err)
	}

	return &metricsExporter{
		cfg:    cfg,
		logger: params.Logger,
		db:     db,
	}, nil
}

func (me *metricsExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (me *metricsExporter) start(ctx context.Context, _ component.Host) error {
	err := me.db.InitDatabaseConn(ctx, me.cfg.Role)
	if err != nil {
		return fmt.Errorf("failed to initialize database connection for metrics: %w", err)
	}

	err = me.db.CreateSchema(ctx, me.cfg.Metrics.Schema)
	if err != nil {
		return fmt.Errorf("failed to create metrics schema: %w", err)
	}

	err = me.createMetricTables(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (me *metricsExporter) createMetricTables(ctx context.Context) error {
	err := me.db.CreateTable(ctx, fmt.Sprintf(metrics.CreateSumMetricTableTemplate, me.cfg.Database, me.cfg.Metrics.Schema, me.cfg.Metrics.Table))
	if err != nil {
		return fmt.Errorf("failed to create sum metrics table: %w", err)
	}

	err = me.db.CreateTable(ctx, fmt.Sprintf(metrics.CreateGaugeMetricTableTemplate, me.cfg.Database, me.cfg.Metrics.Schema, me.cfg.Metrics.Table))
	if err != nil {
		return fmt.Errorf("failed to create gauge metrics table: %w", err)
	}

	err = me.db.CreateTable(ctx, fmt.Sprintf(metrics.CreateSummaryMetricTableTemplate, me.cfg.Database, me.cfg.Metrics.Schema, me.cfg.Metrics.Table))
	if err != nil {
		return fmt.Errorf("failed to create summary metrics table: %w", err)
	}

	err = me.db.CreateTable(ctx, fmt.Sprintf(metrics.CreateHistogramMetricTableTemplate, me.cfg.Database, me.cfg.Metrics.Schema, me.cfg.Metrics.Table))
	if err != nil {
		return fmt.Errorf("failed to create histogram metrics table: %w", err)
	}

	err = me.db.CreateTable(ctx, fmt.Sprintf(metrics.CreateExponentialHistogramMetricTableTemplate, me.cfg.Database, me.cfg.Metrics.Schema, me.cfg.Metrics.Table))
	if err != nil {
		return fmt.Errorf("failed to create exponential histogram metrics table: %w", err)
	}

	return nil
}

func (me *metricsExporter) shutdown(_ context.Context) error {
	if me.db != nil {
		return me.db.Close()
	}
	return nil
}

func (me *metricsExporter) metricsDataPusher(ctx context.Context, md pmetric.Metrics) error {
	me.logger.Debug("begin metricsDataPusher")

	models := me.filterMetrics(md)

	// call BatchInsert for all metric models
	wg := &sync.WaitGroup{}
	errorChan := make(chan error, len(models))
	for _, v := range models {
		wg.Add(1)
		go func(m metricModel) {
			defer wg.Done()
			errorChan <- m.BatchInsert(ctx, me.db)
		}(v)
	}

	go func() {
		wg.Wait()
		close(errorChan)
	}()

	// return any errors
	var errs error
	for e := range errorChan {
		errs = errors.Join(errs, e)
	}

	me.logger.Debug("end metricsDataPusher")

	return errs
}

func (me *metricsExporter) filterMetrics(md pmetric.Metrics) map[string]metricModel {
	m := me.newMetricModels()

	// loop through metrics and add to corresponding metric model
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		resourceMetric := md.ResourceMetrics().At(i)

		for j := 0; j < resourceMetric.ScopeMetrics().Len(); j++ {
			scopeMetric := resourceMetric.ScopeMetrics().At(j)

			for k := 0; k < scopeMetric.Metrics().Len(); k++ {
				metric := scopeMetric.Metrics().At(k)

				switch metric.Type() {
				case pmetric.MetricTypeSum:
					m["sums"].AddMetric(resourceMetric, scopeMetric, metric)
				case pmetric.MetricTypeGauge:
					m["gauges"].AddMetric(resourceMetric, scopeMetric, metric)
				case pmetric.MetricTypeSummary:
					m["summaries"].AddMetric(resourceMetric, scopeMetric, metric)
				case pmetric.MetricTypeHistogram:
					m["histograms"].AddMetric(resourceMetric, scopeMetric, metric)
				case pmetric.MetricTypeExponentialHistogram:
					m["exponentialHistograms"].AddMetric(resourceMetric, scopeMetric, metric)
				default:
					me.logger.Warn("unsupported metric type", zap.String("type", metric.Type().String()))
				}
			}
		}
	}

	return m
}

func (me *metricsExporter) newMetricModels() map[string]metricModel {
	m := map[string]metricModel{}
	m["sums"] = metrics.NewSumModel(me.logger, fmt.Sprintf(metrics.InsertIntoSumMetricTableTemplate, me.cfg.Database, me.cfg.Metrics.Schema, me.cfg.Metrics.Table))
	m["gauges"] = metrics.NewGaugeModel(me.logger, fmt.Sprintf(metrics.InsertIntoGaugeMetricTableTemplate, me.cfg.Database, me.cfg.Metrics.Schema, me.cfg.Metrics.Table))
	m["summaries"] = metrics.NewSummaryModel(me.logger, fmt.Sprintf(metrics.InsertIntoSummaryMetricTableTemplate, me.cfg.Database, me.cfg.Metrics.Schema, me.cfg.Metrics.Table))
	m["histograms"] = metrics.NewHistogramModel(me.logger, fmt.Sprintf(metrics.InsertIntoHistogramMetricTableTemplate, me.cfg.Database, me.cfg.Metrics.Schema, me.cfg.Metrics.Table))
	m["exponentialHistograms"] = metrics.NewExponentialHistogramModel(me.logger, fmt.Sprintf(metrics.InsertIntoExponentialHistogramMetricTableTemplate, me.cfg.Database, me.cfg.Metrics.Schema, me.cfg.Metrics.Table))
	return m
}
