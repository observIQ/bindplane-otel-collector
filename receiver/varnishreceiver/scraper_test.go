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
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied
// See the License for the specific language governing permissions and
// limitations under the License.

package varnishreceiver // import "github.com/observiq/observiq-otel-collector/receiver/varnishreceiver"

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/model/pdata"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestScrape(t *testing.T) {
	f := NewFactory()
	cfg := f.CreateDefaultConfig().(*Config)
	require.NotNil(t, cfg)

	t.Run("test success >= 6.5", func(t *testing.T) {
		mockClient := new(mockClient)
		mockClient.On("GetStats").Return(getStats(t, "mock_response6_5.json"))

		scraper := newVarnishScraper(componenttest.NewNopTelemetrySettings(), cfg)
		scraper.client = mockClient
		actualMetrics, err := scraper.scrape(context.Background())
		require.NoError(t, err)
		require.Equal(t, 1, actualMetrics.ResourceMetrics().Len())

		validateScraperResult(t, actualMetrics)
	})

	t.Run("test success < 6.5", func(t *testing.T) {
		mockClient := new(mockClient)
		mockClient.On("GetStats").Return(getStats(t, "mock_response6_0.json"))

		scraper := newVarnishScraper(componenttest.NewNopTelemetrySettings(), cfg)
		scraper.client = mockClient
		actualMetrics, err := scraper.scrape(context.Background())
		require.NoError(t, err)
		require.Equal(t, 1, actualMetrics.ResourceMetrics().Len())

		validateScraperResult(t, actualMetrics)
	})

	t.Run("scrape error", func(t *testing.T) {
		obs, logs := observer.New(zap.ErrorLevel)
		settings := componenttest.NewNopTelemetrySettings()
		settings.Logger = zap.New(obs)
		mockClient := new(mockClient)
		mockClient.On("GetStats").Return(getStats(t, ""))
		scraper := newVarnishScraper(settings, cfg)
		scraper.client = mockClient

		_, err := scraper.scrape(context.Background())
		require.NotNil(t, err)
		require.Equal(t, 1, logs.Len())
		require.Equal(t, []observer.LoggedEntry{
			{
				Entry: zapcore.Entry{Level: zap.ErrorLevel, Message: "Failed to execute varnishstat"},
				Context: []zapcore.Field{
					zap.String("Cache Dir:", cfg.CacheDir),
					zap.String("Executable Directory:", cfg.ExecDir),
					zap.Error(errors.New("bad response")),
				},
			},
		}, logs.AllUntimed())
	})

}

func validateScraperResult(t *testing.T, actualMetrics pdata.Metrics) {
	require.Equal(t, actualMetrics.MetricCount(), 11)
	require.Equal(t, actualMetrics.DataPointCount(), 26)

	ilms := actualMetrics.ResourceMetrics().At(0).InstrumentationLibraryMetrics()
	require.Equal(t, 1, ilms.Len())
	ms := ilms.At(0).Metrics()
	for i := 0; i < ms.Len(); i++ {
		m := ms.At(i)
		switch m.Name() {
		case "varnish.backend.connection.count":
			dps := m.Sum().DataPoints()
			require.Equal(t, 7, dps.Len())
			attributeMappings := map[string]int64{}
			for j := 0; j < dps.Len(); j++ {
				dp := dps.At(j)
				method := dp.Attributes().AsRaw()
				label := fmt.Sprintf("%s method:%s", m.Name(), method)
				attributeMappings[label] = dp.IntVal()
			}
			require.Equal(t, map[string]int64{
				"varnish.backend.connection.count method:map[kind:busy]":      int64(9),
				"varnish.backend.connection.count method:map[kind:fail]":      int64(10),
				"varnish.backend.connection.count method:map[kind:recycle]":   int64(12),
				"varnish.backend.connection.count method:map[kind:retry]":     int64(13),
				"varnish.backend.connection.count method:map[kind:reuse]":     int64(11),
				"varnish.backend.connection.count method:map[kind:success]":   int64(7),
				"varnish.backend.connection.count method:map[kind:unhealthy]": int64(8),
			},
				attributeMappings)
		case "varnish.cache.operation.count":
			dps := m.Sum().DataPoints()
			require.Equal(t, 3, dps.Len())
			attributeMappings := map[string]int64{}
			for j := 0; j < dps.Len(); j++ {
				dp := dps.At(j)
				method := dp.Attributes().AsRaw()
				label := fmt.Sprintf("%s method:%s", m.Name(), method)
				attributeMappings[label] = dp.IntVal()
			}
			require.Equal(t, map[string]int64{
				"varnish.cache.operation.count method:map[operation:hit]":      int64(4),
				"varnish.cache.operation.count method:map[operation:hit_pass]": int64(5),
				"varnish.cache.operation.count method:map[operation:miss]":     int64(6),
			},
				attributeMappings)
		case "varnish.thread.operation.count":
			dps := m.Sum().DataPoints()
			require.Equal(t, 3, dps.Len())
			attributeMappings := map[string]int64{}
			for j := 0; j < dps.Len(); j++ {
				dp := dps.At(j)
				method := dp.Attributes().AsRaw()
				label := fmt.Sprintf("%s method:%s", m.Name(), method)
				attributeMappings[label] = dp.IntVal()
			}
			require.Equal(t, map[string]int64{
				"varnish.thread.operation.count method:map[operation:created]":   int64(14),
				"varnish.thread.operation.count method:map[operation:destroyed]": int64(15),
				"varnish.thread.operation.count method:map[operation:failed]":    int64(16),
			},
				attributeMappings)
		case "varnish.session.count":
			dps := m.Sum().DataPoints()
			require.Equal(t, 3, dps.Len())
			attributeMappings := map[string]int64{}
			for j := 0; j < dps.Len(); j++ {
				dp := dps.At(j)
				method := dp.Attributes().AsRaw()
				label := fmt.Sprintf("%s method:%s", m.Name(), method)
				attributeMappings[label] = dp.IntVal()
			}
			require.Equal(t, map[string]int64{
				"varnish.session.count method:map[kind:accepted]": int64(1),
				"varnish.session.count method:map[kind:dropped]":  int64(17),
				"varnish.session.count method:map[kind:failed]":   int64(2),
			},
				attributeMappings)
		case "varnish.object.nuked":
			dps := m.Sum().DataPoints()
			require.Equal(t, 1, dps.Len())
			require.EqualValues(t, int64(20), dps.At(0).IntVal())
		case "varnish.object.moved":
			dps := m.Sum().DataPoints()
			require.Equal(t, 1, dps.Len())
			require.EqualValues(t, int64(21), dps.At(0).IntVal())
		case "varnish.object.expired":
			dps := m.Sum().DataPoints()
			require.Equal(t, 1, dps.Len())
			require.EqualValues(t, int64(19), dps.At(0).IntVal())
		case "varnish.object.count":
			dps := m.Sum().DataPoints()
			require.Equal(t, 1, dps.Len())
			require.EqualValues(t, int64(18), dps.At(0).IntVal())
		case "varnish.client.request.count":
			dps := m.Sum().DataPoints()
			require.Equal(t, 2, dps.Len())
			attributeMappings := map[string]int64{}
			for j := 0; j < dps.Len(); j++ {
				dp := dps.At(j)
				method := dp.Attributes().AsRaw()
				label := fmt.Sprintf("%s method:%s", m.Name(), method)
				attributeMappings[label] = dp.IntVal()
			}
			require.Equal(t, map[string]int64{
				"varnish.client.request.count method:map[state:received]": int64(3),
				"varnish.client.request.count method:map[state:dropped]":  int64(23),
			},
				attributeMappings)
		case "varnish.client.request.error.count":
			dps := m.Sum().DataPoints()
			require.Equal(t, 3, dps.Len())
			attributeMappings := map[string]int64{}
			for j := 0; j < dps.Len(); j++ {
				dp := dps.At(j)
				method := dp.Attributes().AsRaw()
				label := fmt.Sprintf("%s method:%s", m.Name(), method)
				attributeMappings[label] = dp.IntVal()
			}
			require.Equal(t, map[string]int64{
				"varnish.client.request.error.count method:map[status_code:400]": int64(24),
				"varnish.client.request.error.count method:map[status_code:417]": int64(25),
				"varnish.client.request.error.count method:map[status_code:500]": int64(26),
			},
				attributeMappings)
		case "varnish.backend.request.count":
			dps := m.Sum().DataPoints()
			require.Equal(t, 1, dps.Len())
			require.EqualValues(t, int64(22), dps.At(0).IntVal())
		}
	}
}

func TestStart(t *testing.T) {
	t.Run("start with default hostname", func(t *testing.T) {
		hostname, err := os.Hostname()
		require.Nil(t, err)

		f := NewFactory()
		cfg := f.CreateDefaultConfig().(*Config)
		scraper := newVarnishScraper(componenttest.NewNopTelemetrySettings(), cfg)
		err = scraper.start(context.Background(), componenttest.NewNopHost())
		require.NoError(t, err)
		require.EqualValues(t, hostname, scraper.cacheName)
	})
	t.Run("start with specified cache dir", func(t *testing.T) {
		f := NewFactory()
		cfg := f.CreateDefaultConfig().(*Config)
		cfg.CacheDir = "/path/cache_name"
		scraper := newVarnishScraper(componenttest.NewNopTelemetrySettings(), cfg)
		err := scraper.start(context.Background(), componenttest.NewNopHost())
		require.NoError(t, err)
		require.EqualValues(t, "cache_name", scraper.cacheName)
	})
}

func TestSetDefaultCacheName(t *testing.T) {
	t.Run("missing cache dir", func(t *testing.T) {
		f := NewFactory()
		cfg := f.CreateDefaultConfig().(*Config)
		scraper := newVarnishScraper(componenttest.NewNopTelemetrySettings(), cfg)

		hostname, err := os.Hostname()
		require.NoError(t, err)

		err = scraper.setCacheName()
		require.NoError(t, err)
		require.EqualValues(t, hostname, scraper.cacheName)
	})

	t.Run("found cache dir", func(t *testing.T) {
		f := NewFactory()
		cfg := f.CreateDefaultConfig().(*Config)
		cfg.CacheDir = "/path/cache_name"
		scraper := newVarnishScraper(componenttest.NewNopTelemetrySettings(), cfg)

		err := scraper.setCacheName()
		require.NoError(t, err)
		require.EqualValues(t, "cache_name", scraper.cacheName)
	})
}

func getStats(t *testing.T, filename string) (*Stats, error) {
	t.Helper()
	if filename == "" {
		return nil, errors.New("bad response")
	}

	body, err := os.ReadFile(path.Join("testdata", "scraper", filename))
	if err != nil {
		return nil, err
	}

	return parseStats(body)
}

// mockClient is an autogenerated mock type for the mockClient type
type mockClient struct {
	mock.Mock
}

// GetStats provides a mock function with given fields:
func (_m *mockClient) GetStats() (*Stats, error) {
	ret := _m.Called()

	var r0 *Stats
	if rf, ok := ret.Get(0).(func() *Stats); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*Stats)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
