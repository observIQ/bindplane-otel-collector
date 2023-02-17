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

package aggregationprocessor

import (
	"context"
	"encoding/binary"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/observiq/observiq-otel-collector/processor/aggregationprocessor/internal/aggregate"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type aggregationProcessor struct {
	logger   *zap.Logger
	mux      sync.Mutex
	wg       sync.WaitGroup
	doneChan chan struct{}
	//for mocking in test
	now func() time.Time

	includeRegex           *regexp.Regexp
	flushInterval          time.Duration
	aggregationPeriodStart pcommon.Timestamp
	aggregationTypes       []aggregate.AggregationType
	// map resource hash to resourceAggregation
	aggregationMap map[uint64]*resourceMetadata
	nextConsumer   consumer.Metrics
}

func newAggregationProcessor(logger *zap.Logger, cfg *Config, consumer consumer.Metrics) (*aggregationProcessor, error) {
	regex, err := regexp.Compile(cfg.Include)
	if err != nil {
		return nil, fmt.Errorf("failed to compile include regex: %w", err)
	}
	return &aggregationProcessor{
		logger:                 logger,
		mux:                    sync.Mutex{},
		wg:                     sync.WaitGroup{},
		doneChan:               make(chan struct{}),
		now:                    time.Now,
		includeRegex:           regex,
		flushInterval:          cfg.Interval,
		aggregationPeriodStart: pcommon.NewTimestampFromTime(time.Now()),
		aggregationMap:         make(map[uint64]*resourceMetadata),
		aggregationTypes:       cfg.AggregationTypes(),
		nextConsumer:           consumer,
	}, nil
}

func (sp *aggregationProcessor) Start(_ context.Context, _ component.Host) error {
	sp.wg.Add(1)
	go sp.flushLoop()
	return nil
}

func (sp *aggregationProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	sp.aggregateMetrics(md)
	if md.ResourceMetrics().Len() != 0 {
		// Forward metrics we didn't consume
		return sp.nextConsumer.ConsumeMetrics(ctx, md)
	}

	return nil
}

// Add metrics that we care about to our aggregation.
// The incoming pmetric.Metrics is modified, such that aggregated metrics are removed.
func (sp *aggregationProcessor) aggregateMetrics(md pmetric.Metrics) {
	sp.mux.Lock()
	defer sp.mux.Unlock()

	rms := md.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		resAttrs := rm.Resource().Attributes()
		resKey := mapKey(resAttrs)

		sms := rm.ScopeMetrics()
		for j := 0; j < sms.Len(); j++ {
			sm := sms.At(j)
			ms := sm.Metrics()
			for k := 0; k < ms.Len(); k++ {
				m := ms.At(k)
				if !canAggregateMetric(m) {
					continue
				}

				// Metric must match regex
				if !sp.includeRegex.MatchString(m.Name()) {
					continue
				}

				ma := sp.metricAggregation(m, resKey, resAttrs)

				dps := datapointsFromMetric(m)
				// We remove datapoints that we aggregate here, so we use RemoveIf to iterate the datapoints
				dps.RemoveIf(func(dp pmetric.NumberDataPoint) bool {
					if dp.ValueType() != pmetric.NumberDataPointValueTypeDouble &&
						dp.ValueType() != pmetric.NumberDataPointValueTypeInt {
						// Ignore values that are not Double or Int (e.g. are empty)
						return false
					}

					sp.aggregateDatapoint(ma, dp)
					return true
				})
			}

			// remove the metric if we consumed all the datapoints
			removeEmptyMetrics(ms)
		}
		// remove the scope metrics if we consumed all the metrics
		removeEmptyScopeMetrics(sms)
	}
	// remove the resource metrics if we consumed all the ScopeMetrics
	removeEmptyResourceMetrics(rms)
}

// metricAggregation gets the metricAggregation for the given metric & resource, creating it if it doesn't exist.
func (sp *aggregationProcessor) metricAggregation(m pmetric.Metric, resKey uint64, resAttrs pcommon.Map) *metricMetadata {
	rma, ok := sp.aggregationMap[resKey]
	if !ok {
		// Track the resource information for this resource if we haven't already
		rma = &resourceMetadata{
			resource: resAttrs,
			metrics:  make(map[string]*metricMetadata),
		}
		sp.aggregationMap[resKey] = rma
	}

	ma, ok := rma.metrics[m.Name()]
	if !ok {
		// Track the metadata for this metric if we haven't already.
		ma = &metricMetadata{
			name:       m.Name(),
			desc:       m.Description(),
			unit:       m.Unit(),
			metricType: m.Type(),
			monotonic:  isMonotonic(m),
			datapoints: make(map[uint64]*datapointMetadata),
		}
		rma.metrics[m.Name()] = ma
	}

	return ma
}

// aggregateDatapoint either adds the datapoint to an existing aggregate (if one exists for the NumberDataPoint's attributes),
// or creates a new aggregate for the datapoint.
func (sp *aggregationProcessor) aggregateDatapoint(ma *metricMetadata, dp pmetric.NumberDataPoint) {
	attributeKey := mapKey(dp.Attributes())
	dpa, ok := ma.datapoints[attributeKey]
	if !ok {
		// Create the aggregations for this datapoint if we haven't already for this set of attributes.
		aggs, err := sp.createAggregates(dp)
		if err != nil {
			sp.logger.Error("Failed to create some aggregates.", zap.Error(err), zap.String("metric", ma.name))
			// We continue here even if some aggregates failed to be created
		}

		dpa = &datapointMetadata{
			attributes: dp.Attributes(),
			aggregates: aggs,
		}
		ma.datapoints[attributeKey] = dpa

		// we don't need to call AddDatapoint, since the aggregates are initialized with the first datapoint.
		return
	}

	// Add datapoints to existing aggregates
	for _, agg := range dpa.aggregates {
		agg.AddDatapoint(dp)
	}
}

// createAggregates creates all aggregates for this datapoint based on the configuration of this processor
// The returned error here is a multierr, and may be a partial err, so the resultant map may be used even if an error is returned.
func (sp *aggregationProcessor) createAggregates(initialVal pmetric.NumberDataPoint) (map[aggregate.AggregationType]aggregate.Aggregate, error) {
	var errs error
	aggs := make(map[aggregate.AggregationType]aggregate.Aggregate, len(sp.aggregationTypes))
	for _, aggType := range sp.aggregationTypes {
		agg, err := aggType.New(initialVal)
		if err != nil {
			errs = multierr.Append(errs, fmt.Errorf("failed to create aggregation: %w", err))
			continue
		}

		aggs[aggType] = agg
	}

	return aggs, errs
}

// flushLoop is a goroutine that flushes all aggregates every sp.flushInterval.
func (sp *aggregationProcessor) flushLoop() {
	defer sp.wg.Done()

	t := time.NewTicker(sp.flushInterval)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			sp.flush()
		case <-sp.doneChan:
			return
		}
	}
}

// flush flushes all aggregations to the next component in the collector pipeline.
func (sp *aggregationProcessor) flush() {
	sp.mux.Lock()
	defer sp.mux.Unlock()

	now := pcommon.NewTimestampFromTime(sp.now())
	metrics := pmetric.NewMetrics()

	for _, ra := range sp.aggregationMap {
		rm := metrics.ResourceMetrics().AppendEmpty()
		ra.resource.CopyTo(rm.Resource().Attributes())
		sm := rm.ScopeMetrics().AppendEmpty()

		for _, aggConf := range sp.aggregationTypes {
			for _, ma := range ra.metrics {
				sp.addAggregateMetric(now, sm.Metrics(), ma, aggConf)
			}
		}
	}

	if metrics.DataPointCount() != 0 {
		if err := sp.nextConsumer.ConsumeMetrics(context.Background(), metrics); err != nil {
			sp.logger.Error("Failed to consume metrics.", zap.Error(err))
		}
	}

	// Reset aggregation map
	sp.aggregationMap = make(map[uint64]*resourceMetadata)

	// Aggregation period will start from when we started flush.
	sp.aggregationPeriodStart = now
}

func (sp *aggregationProcessor) addAggregateMetric(now pcommon.Timestamp, ms pmetric.MetricSlice, ma *metricMetadata, aggType aggregate.AggregationType) {
	m := ms.AppendEmpty()

	m.SetName(fmt.Sprintf("%s.%s", ma.name, aggType))
	m.SetDescription(ma.desc)
	m.SetUnit(ma.unit)

	var dps pmetric.NumberDataPointSlice
	switch ma.metricType {
	case pmetric.MetricTypeGauge:
		g := m.SetEmptyGauge()
		dps = g.DataPoints()
	case pmetric.MetricTypeSum:
		s := m.SetEmptySum()
		s.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		s.SetIsMonotonic(ma.monotonic)
		dps = s.DataPoints()
	}

	for _, dpa := range ma.datapoints {
		agg, ok := dpa.aggregates[aggType]
		if !ok {
			// aggregation must have failed to be created, so we can't emit this as a metric
			continue
		}

		// Construct datapoints
		dp := dps.AppendEmpty()
		agg.SetDatapointValue(dp)
		dpa.attributes.CopyTo(dp.Attributes())
		dp.SetStartTimestamp(sp.aggregationPeriodStart)
		dp.SetTimestamp(now)
	}
}

func (sp *aggregationProcessor) Capabilities() consumer.Capabilities {
	// Data is mutate, since we remove Metric payloads if they are aggregated
	return consumer.Capabilities{MutatesData: true}
}

func (sp *aggregationProcessor) Shutdown(ctx context.Context) error {
	close(sp.doneChan)

	waitDoneChan := make(chan struct{})
	// wait in a goroutine so that we can select on context cancellation as well
	go func() {
		sp.wg.Wait()
		close(waitDoneChan)
	}()

	select {
	case <-ctx.Done():
		sp.logger.Error("Context timed out while waiting for graceful shutdown.", zap.Error(ctx.Err()))
		return ctx.Err()
	case <-waitDoneChan: // OK
	}

	return nil
}

func canAggregateMetric(m pmetric.Metric) bool {
	switch m.Type() {
	case pmetric.MetricTypeGauge:
		return true
	case pmetric.MetricTypeSum:
		return m.Sum().AggregationTemporality() == pmetric.AggregationTemporalityCumulative
	}

	// Currently only gauges and cumulative sums are supported.
	return false
}

// mapKey returns a unique key for the provided map.
func mapKey(dimension pcommon.Map) uint64 {
	b := pdatautil.MapHash(dimension)
	// Since the hash is 128 bits, and we want a 64 bit hash,
	// we'll condense by XORing the lower and upper 64 bits.
	upper := binary.BigEndian.Uint64(b[:8])
	lower := binary.BigEndian.Uint64(b[8:])
	return lower ^ upper
}
