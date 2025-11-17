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

package resolver

import (
	"context"
	"fmt"
	"net"

	lru "github.com/hashicorp/golang-lru/v2"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

const (
	meterName         = "internal/resolver"
	logFieldHostname  = "hostname"
	logFieldAddresses = "addresses"
)

// Resolver is a DNS resolver that caches DNS lookups using a LRU cache.
// It provides a DialContext function that can be used with http.Transport
// to enable DNS caching for HTTP clients. The resolver is thread-safe.
type Resolver struct {

	// baseResolver is the underlying resolver used for actual DNS lookups
	baseResolver *net.Resolver

	// cache stores DNS lookup results
	// hashicorp/golang-lru is a thread-safe LRU cache implementation
	cache *lru.Cache[string, []net.IPAddr]

	// metrics
	cacheSizeGauge     metric.Int64ObservableGauge
	cacheCapacityGauge metric.Int64ObservableGauge
	cacheHitsCounter   metric.Int64Counter
	cacheMissesCounter metric.Int64Counter

	logger *zap.Logger
}

// New creates a new cached DNS resolver. New requires a MeterProvider, Logger, and cache capacity.
func New(mp metric.MeterProvider, logger *zap.Logger, cacheCapacity int) (*Resolver, error) {
	if logger == nil {
		return nil, fmt.Errorf("Logger is required")
	}

	if mp == nil {
		return nil, fmt.Errorf("MeterProvider is required")
	}

	if cacheCapacity <= 0 {
		return nil, fmt.Errorf("cache size must be greater than 0, got %d", cacheCapacity)
	}

	cache, err := lru.New[string, []net.IPAddr](cacheCapacity)
	if err != nil {
		return nil, fmt.Errorf("failed to create LRU cache: %w", err)
	}

	r := &Resolver{
		baseResolver: net.DefaultResolver,
		cache:        cache,
		logger:       logger,
	}

	if err := r.initMetrics(mp, cacheCapacity); err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}

	return r, nil
}

// initMetrics initializes OpenTelemetry metrics for the resolver.
func (r *Resolver) initMetrics(mp metric.MeterProvider, cacheCapacity int) error {
	meter := mp.Meter(meterName)

	cacheSizeGauge, err := meter.Int64ObservableGauge(
		"dns_resolver_cache_size",
		metric.WithDescription("Current number of entries in the DNS resolver cache"),
		metric.WithUnit("{entries}"),
	)
	if err != nil {
		return fmt.Errorf("create cache_size gauge: %w", err)
	}
	r.cacheSizeGauge = cacheSizeGauge

	cacheCapacityGauge, err := meter.Int64ObservableGauge(
		"dns_resolver_cache_capacity",
		metric.WithDescription("Maximum number of entries the DNS resolver cache can hold"),
		metric.WithUnit("{entries}"),
	)
	if err != nil {
		return fmt.Errorf("create cache_capacity gauge: %w", err)
	}
	r.cacheCapacityGauge = cacheCapacityGauge

	cacheHitsCounter, err := meter.Int64Counter(
		"dns_resolver_cache_fallback_hits",
		metric.WithDescription("Number of DNS lookups that fell back to cached results after lookup failure"),
		metric.WithUnit("{hits}"),
	)
	if err != nil {
		return fmt.Errorf("create cache_fallback_hits counter: %w", err)
	}
	r.cacheHitsCounter = cacheHitsCounter

	cacheMissesCounter, err := meter.Int64Counter(
		"dns_resolver_cache_fallback_misses",
		metric.WithDescription("Number of DNS lookups that failed with no cached result available for fallback"),
		metric.WithUnit("{misses}"),
	)
	if err != nil {
		return fmt.Errorf("create cache_fallback_misses counter: %w", err)
	}
	r.cacheMissesCounter = cacheMissesCounter

	_, err = meter.RegisterCallback(
		func(ctx context.Context, o metric.Observer) error {
			o.ObserveInt64(r.cacheSizeGauge, int64(r.cache.Len()))
			o.ObserveInt64(r.cacheCapacityGauge, int64(cacheCapacity))
			return nil
		},
		r.cacheSizeGauge,
		r.cacheCapacityGauge,
	)
	if err != nil {
		return fmt.Errorf("register metric callbacks: %w", err)
	}

	return nil
}

// DialContext provides a function that can be used as the DialContext for http.Transport.
// It performs DNS lookups using the cached resolver and dials the resulting addresses.
// This allows HTTP clients to benefit from DNS caching, reducing DNS query load
// and improving performance for repeated requests to the same hosts.
//
// Example usage with HTTP client:
//
//	resolver, _ := resolver.New(meterProvider, logger, 1000)
//	client := &http.Client{
//		Transport: &http.Transport{
//			DialContext: resolver.DialContext,
//		},
//	}
func (r *Resolver) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	r.logger.Debug("Dialing new connection", zap.String("network", network), zap.String("address", address))

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("split host and port for address %s: %w", address, err)
	}

	addrs, err := r.lookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("lookup IP addresses for host %s: %w", host, err)
	}

	if len(addrs) == 0 {
		return nil, fmt.Errorf("no addresses found for host: %s", host)
	}

	var lastErr error
	for _, addr := range addrs {
		dialer := &net.Dialer{}
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(addr.IP.String(), port))
		if err == nil {
			r.logger.Debug("Dialed successfully", zap.String("address", addr.IP.String()))
			return conn, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("failed to dial %s: %w", address, lastErr)
}

// lookupIPAddr looks up host using the local resolver and returns a slice of
// IP addresses. It always attempts a fresh lookup first, caching the result if successful.
// If the lookup fails, it returns the cached result if available, otherwise returns the lookup error.
func (r *Resolver) lookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	r.logger.Debug("Looking up IP addresses for host", zap.String("host", host))

	// Always attempt a fresh lookup first
	addrs, err := r.baseResolver.LookupIPAddr(ctx, host)
	if err == nil {
		// Lookup succeeded - cache the result and return it
		r.cache.Add(host, addrs)
		r.logger.Debug("DNS lookup succeeded and cached", zap.String(logFieldHostname, host), zap.Any(logFieldAddresses, addrs))
		return addrs, nil
	}

	// Lookup failed - check if we have a cached result
	cached, ok := r.cache.Get(host)
	if ok {
		// Return cached result as fallback
		r.cacheHitsCounter.Add(ctx, 1)
		r.logger.Debug("DNS lookup failed, using cached result", zap.String(logFieldHostname, host), zap.Any(logFieldAddresses, cached), zap.Error(err))
		return cached, nil
	}

	// No cache entry available - return the lookup error
	r.cacheMissesCounter.Add(ctx, 1)
	r.logger.Debug("DNS lookup failed and no cache entry available", zap.String(logFieldHostname, host), zap.Error(err))
	return nil, fmt.Errorf("lookup IP address for host %s: %w", host, err)
}
