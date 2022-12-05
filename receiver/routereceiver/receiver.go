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

package routereceiver

import (
	"context"
	"errors"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var (
	// routes is a map of registered routes.
	routes = map[string]*receiver{}

	// mux is a mutex for accessing receivers.
	mux sync.RWMutex

	// errMetricPipelineNotDefined is returned when a metric pipeline is not set.
	errMetricPipelineNotDefined = errors.New("metric pipeline not defined for route")

	// errLogPipelineNotDefined is returned when a log pipeline is not set.
	errLogPipelineNotDefined = errors.New("log pipeline not defined for route")

	// errTracePipelineNotDefined is returned when a trace pipeline is not set.
	errTracePipelineNotDefined = errors.New("trace pipeline not defined for route")

	// errReceiverNotSet is an error returned when a receiver is not set.
	errRouteNotDefined = errors.New("route not defined")
)

// receiver is a struct that receives routed telemetry.
type receiver struct {
	name           string
	metricConsumer consumer.Metrics
	logConsumer    consumer.Logs
	traceConsumer  consumer.Traces
}

// Start starts the receiver.
func (r *receiver) Start(_ context.Context, _ component.Host) error {
	return nil
}

// Shutdown stops the receiver.
func (r *receiver) Shutdown(_ context.Context) error {
	removeRoute(r.name)
	return nil
}

// registerMetricConsumer registers a metric consumer.
func (r *receiver) registerMetricConsumer(consumer consumer.Metrics) {
	r.metricConsumer = consumer
}

// registerLogConsumer registers a log consumer.
func (r *receiver) registerLogConsumer(consumer consumer.Logs) {
	r.logConsumer = consumer
}

// registerTraceConsumer registers a trace consumer.
func (r *receiver) registerTraceConsumer(consumer consumer.Traces) {
	r.traceConsumer = consumer
}

// consumeMetrics consumes incoming metrics.
func (r *receiver) consumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	if r.metricConsumer == nil {
		return errMetricPipelineNotDefined
	}

	return r.metricConsumer.ConsumeMetrics(ctx, md)
}

// consumeLogs consumes incoming logs.
func (r *receiver) consumeLogs(ctx context.Context, ld plog.Logs) error {
	if r.logConsumer == nil {
		return errLogPipelineNotDefined
	}

	return r.logConsumer.ConsumeLogs(ctx, ld)
}

// consumeTraces consumes incoming traces.
func (r *receiver) consumeTraces(ctx context.Context, td ptrace.Traces) error {
	if r.traceConsumer == nil {
		return errTracePipelineNotDefined
	}

	return r.traceConsumer.ConsumeTraces(ctx, td)
}

// newReceiver creates a new route receiver.
func newReceiver(name string) *receiver {
	return &receiver{
		name: name,
	}
}

// RouteMetrics routes metrics to a registered route.
func RouteMetrics(ctx context.Context, name string, md pmetric.Metrics) error {
	route, ok := getRoute(name)
	if !ok {
		return errRouteNotDefined
	}

	return route.consumeMetrics(ctx, md)
}

// RouteLogs routes logs to a registered route.
func RouteLogs(ctx context.Context, name string, ld plog.Logs) error {
	route, ok := getRoute(name)
	if !ok {
		return errRouteNotDefined
	}

	return route.consumeLogs(ctx, ld)
}

// RouteTraces routes traces to a registered route.
func RouteTraces(ctx context.Context, name string, td ptrace.Traces) error {
	route, ok := getRoute(name)
	if !ok {
		return errRouteNotDefined
	}

	return route.consumeTraces(ctx, td)
}

// createOrGetRoute creates a new route or returns an existing route.
func createOrGetRoute(name string) *receiver {
	mux.Lock()
	defer mux.Unlock()

	if r, ok := routes[name]; ok {
		return r
	}

	r := newReceiver(name)
	routes[name] = r

	return r
}

// getRoute returns a route by name.
func getRoute(name string) (*receiver, bool) {
	mux.RLock()
	defer mux.RUnlock()

	route, ok := routes[name]
	return route, ok
}

// removeRoute removes a route by name.
func removeRoute(name string) {
	mux.Lock()
	defer mux.Unlock()

	delete(routes, name)
}
