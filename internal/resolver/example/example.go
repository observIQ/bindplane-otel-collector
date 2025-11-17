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

// Package main demonstrates how to use the caching DNS resolver with HTTP clients.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/observiq/bindplane-otel-collector/internal/resolver"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.uber.org/zap"
)

// NewHTTPClient creates a new HTTP client that uses a cached DNS resolver.
// The resolver caches DNS lookups to improve performance and reduce DNS query load.
func NewHTTPClient(mp otelmetric.MeterProvider, logger *zap.Logger, cacheCapacity int) (*http.Client, error) {
	r, err := resolver.New(mp, logger.Named("Resolver"), cacheCapacity)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext:       r.DialContext,
			Proxy:             http.ProxyFromEnvironment,
			ForceAttemptHTTP2: true,
			MaxIdleConns:      100,

			// Idle timeout is less than the 10 second ticker, so we should see
			// a new connection each time
			IdleConnTimeout:       2 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	return client, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <host>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s google.com\n", os.Args[0])
		os.Exit(1)
	}
	host := os.Args[1]

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	config.Encoding = "json"
	logger, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("Failed to create logger: %v", err))
	}
	defer logger.Sync()
	logger = logger.Named("main")

	mp := metric.NewMeterProvider()
	defer mp.Shutdown(context.Background())

	client, err := NewHTTPClient(mp, logger, 1000)
	if err != nil {
		logger.Fatal("Failed to create HTTP client", zap.Error(err))
	}

	urlStr := host
	if len(host) >= 7 && host[:7] == "http://" {
	} else if len(host) >= 8 && host[:8] == "https://" {
	} else {
		urlStr = "https://" + host
	}

	ctx := context.Background()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
		if err != nil {
			logger.Error("Failed to create request", zap.String("url", urlStr), zap.Error(err))
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			logger.Error("Failed to make request", zap.String("url", urlStr), zap.Error(err))
			continue
		}

		resp.Body.Close()
	}
}
