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

package sapnetweaverreceiver // import "github.com/observiq/observiq-otel-collector/receiver/sapnetweaverreceiver"

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/observiq/observiq-otel-collector/receiver/sapnetweaverreceiver/internal/metadata"
)

// NewFactory creates a factory for SAP Netweaver receiver.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability))
}

func createDefaultConfig() component.Config {
	return &Config{
		ScraperControllerSettings: scraperhelper.ScraperControllerSettings{
			CollectionInterval: 60 * time.Second,
		},
		HTTPClientSettings: confighttp.HTTPClientSettings{
			Endpoint: defaultEndpoint,
			Timeout:  10 * time.Second,
		},
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
	}
}

var errConfigNotNetweaver = errors.New("config was not an sapnetweaver receiver config")

func createMetricsReceiver(
	_ context.Context,
	params receiver.CreateSettings,
	rConf component.Config,
	consumer consumer.Metrics,
) (receiver.Metrics, error) {
	cfg, ok := rConf.(*Config)
	if !ok {
		return nil, errConfigNotNetweaver
	}

	ns := newSapNetweaverScraper(params, cfg)
	scraper, err := scraperhelper.NewScraper(metadata.Type, ns.scrape, scraperhelper.WithStart(ns.start))
	if err != nil {
		return nil, err
	}

	return scraperhelper.NewScraperControllerReceiver(
		&cfg.ScraperControllerSettings, params, consumer,
		scraperhelper.AddScraper(scraper),
	)
}
