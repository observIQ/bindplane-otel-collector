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

package metricstatsprocessor

import (
	"context"
	"encoding/binary"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/observiq/bindplane-agent/processor/metricstatsprocessor/internal/stats"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatautil"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

type metricstatsProcessor struct {
	logger   *zap.Logger
	mux      sync.Mutex
	wg       sync.WaitGroup
	doneChan chan struct{}
	//for mocking in test
	now func() time.Time

	includeRegex    *regexp.Regexp
	flushInterval   time.Duration
	calcPeriodStart pcommon.Timestamp
	statTypes       []stats.StatType
	// map resource hash to resourceMetadata
	statMap      map[uint64]*resourceMetadata
	nextConsumer consumer.Metrics
}

func newStatsProcessor(logger *zap.Logger, cfg *Config, consumer consumer.Metrics) (*metricstatsProcessor, error) {
	regex, err := regexp.Compile(cfg.Include)
	if err != nil {
		return nil, fmt.Errorf("failed to compile include regex: %w", err)
	}
	return &metricstatsProcessor{
		logger:          logger,
		mux:             sync.Mutex{},
		wg:              sync.WaitGroup{},
		doneChan:        make(chan struct{}),
		now:             time.Now,
		includeRegex:    regex,
		flushInterval:   cfg.Interval,
		calcPeriodStart: pcommon.NewTimestampFromTime(time.Now()),
		statMap:         make(map[uint64]*resourceMetadata),
		statTypes:       cfg.StatTypes(),
		nextConsumer:    consumer,
	}, nil
}

func (sp *metricstatsProcessor) Start(_ context.Context, _ component.Host) error {
	sp.wg.Add(1)
	go sp.flushLoop()
	return nil
}

func (sp *metricstatsProcessor) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	sp.addMetricsToCalculations(md)
	if md.ResourceMetrics().Len() != 0 {
		// Forward metrics we didn't consume
		return sp.nextConsumer.ConsumeMetrics(ctx, md)
	}

	return nil
}

// Add metrics that are valid and match the include regex to our calculations.
// The incoming pmetric.Metrics is modified, such that matching metrics are removed.
func (sp *metricstatsProcessor) addMetricsToCalculations(md pmetric.Metrics) {
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
				if !canAddMetricToStats(m) {
					continue
				}

				// Metric must match regex
				if !sp.includeRegex.MatchString(m.Name()) {
					continue
				}

				ma := sp.metricMetadata(m, resKey, resAttrs)

				dps := datapointsFromMetric(m)
				// We remove datapoints that we add to our statistics here, so we use RemoveIf to iterate the datapoints
				dps.RemoveIf(func(dp pmetric.NumberDataPoint) bool {
					if dp.ValueType() != pmetric.NumberDataPointValueTypeDouble &&
						dp.ValueType() != pmetric.NumberDataPointValueTypeInt {
						// Ignore values that are not Double or Int (e.g. are empty)
						return false
					}

					sp.addDatapointToStats(ma, dp)
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

// metricMetadata gets the metricMetadata for the given metric & resource, creating it if it doesn't exist.
func (sp *metricstatsProcessor) metricMetadata(m pmetric.Metric, resKey uint64, resAttrs pcommon.Map) *metricMetadata {
	rma, ok := sp.statMap[resKey]
	if !ok {
		// Track the resource information for this resource if we haven't already
		rma = &resourceMetadata{
			resource: resAttrs,
			metrics:  make(map[string]*metricMetadata),
		}
		sp.statMap[resKey] = rma
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

// addDatapointToStats either adds the datapoint to all existing statistics (if one exists for the NumberDataPoint's attributes),
// or creates a new set of statistics for the datapoint.
func (sp *metricstatsProcessor) addDatapointToStats(ma *metricMetadata, dp pmetric.NumberDataPoint) {
	attributeKey := mapKey(dp.Attributes())
	dpa, ok := ma.datapoints[attributeKey]
	if !ok {
		// Create the statistics for this datapoint if we haven't already for this set of attributes.
		statistics, err := sp.createStatistics(dp)
		if err != nil {
			sp.logger.Error("Failed to create some statistics.", zap.Error(err), zap.String("metric", ma.name))
			// We continue here even if some statistics failed to be created
		}

		dpa = &datapointMetadata{
			attributes: dp.Attributes(),
			statistics: statistics,
		}
		ma.datapoints[attributeKey] = dpa

		// we don't need to call AddDatapoint, since the statistics are initialized with the first datapoint.
		return
	}

	// Add datapoints to existing statistics
	for _, stat := range dpa.statistics {
		stat.AddDatapoint(dp)
	}
}

// createStatistics creates all statistics for this datapoint based on the configuration of this processor
// The returned error here is a multierr, and may be a partial err, so the resultant map may be used even if an error is returned.
func (sp *metricstatsProcessor) createStatistics(initialVal pmetric.NumberDataPoint) (map[stats.StatType]stats.Statistic, error) {
	var errs error
	statistics := make(map[stats.StatType]stats.Statistic, len(sp.statTypes))
	for _, statType := range sp.statTypes {
		stat, err := statType.New(initialVal)
		if err != nil {
			errs = multierr.Append(errs, fmt.Errorf("failed to create statistic: %w", err))
			continue
		}

		statistics[statType] = stat
	}

	return statistics, errs
}

// flushLoop is a goroutine that flushes all statistics every sp.flushInterval.
func (sp *metricstatsProcessor) flushLoop() {
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

// flush flushes all statistics to the next component in the collector pipeline.
func (sp *metricstatsProcessor) flush() {
	sp.mux.Lock()
	defer sp.mux.Unlock()

	now := pcommon.NewTimestampFromTime(sp.now())
	metrics := pmetric.NewMetrics()

	for _, ra := range sp.statMap {
		rm := metrics.ResourceMetrics().AppendEmpty()
		ra.resource.CopyTo(rm.Resource().Attributes())
		sm := rm.ScopeMetrics().AppendEmpty()

		for _, statType := range sp.statTypes {
			for _, ma := range ra.metrics {
				sp.addCalculatedMetric(now, sm.Metrics(), ma, statType)
			}
		}
	}

	if metrics.DataPointCount() != 0 {
		if err := sp.nextConsumer.ConsumeMetrics(context.Background(), metrics); err != nil {
			sp.logger.Error("Failed to consume metrics.", zap.Error(err))
		}
	}

	// Reset statistic map
	sp.statMap = make(map[uint64]*resourceMetadata)

	// Calculation period will start from when we started flush.
	sp.calcPeriodStart = now
}

func (sp *metricstatsProcessor) addCalculatedMetric(now pcommon.Timestamp, ms pmetric.MetricSlice, ma *metricMetadata, statType stats.StatType) {
	m := ms.AppendEmpty()

	m.SetName(fmt.Sprintf("%s.%s", ma.name, statType))
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
		stat, ok := dpa.statistics[statType]
		if !ok {
			// this statistics must have failed to be created, so we can't emit this as a metric
			continue
		}

		// Construct datapoints
		dp := dps.AppendEmpty()
		stat.SetDatapointValue(dp)
		dpa.attributes.CopyTo(dp.Attributes())
		dp.SetStartTimestamp(sp.calcPeriodStart)
		dp.SetTimestamp(now)
	}
}

func (sp *metricstatsProcessor) Capabilities() consumer.Capabilities {
	// Data is mutate, since we remove Metric payloads if they are add to a statistic
	return consumer.Capabilities{MutatesData: true}
}

func (sp *metricstatsProcessor) Shutdown(ctx context.Context) error {
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

func canAddMetricToStats(m pmetric.Metric) bool {
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
