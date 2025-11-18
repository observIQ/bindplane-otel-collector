// Copyright observIQ, Inc.
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

package kandjireceiver // import "github.com/observiq/bindplane-otel-collector/receiver/kandjireceiver"

import (
	"context"
	"time"

	"github.com/observiq/bindplane-otel-collector/receiver/kandjireceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/scraper"
	"go.opentelemetry.io/collector/scraper/scraperhelper"
)

// NewFactory creates a factory for the Kandji receiver.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability),
		receiver.WithLogs(createLogsReceiver, component.StabilityLevelAlpha),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		ControllerConfig: scraperhelper.ControllerConfig{
			CollectionInterval: 5 * time.Minute,
		},
		ClientConfig: confighttp.ClientConfig{
			Timeout: 15 * time.Second,
		},

		Region:    "US",
		SubDomain: "",
		ApiKey:    "",
		BaseHost:  "api.kandji.io",

		Logs: LogsConfig{
			PollInterval: 5 * time.Minute,
		},

		StorageID:      nil,
		EndpointParams: map[string]map[string]any{},
	}
}

func createMetricsReceiver(
	_ context.Context,
	params receiver.Settings,
	rConf component.Config,
	cons consumer.Metrics,
) (receiver.Metrics, error) {

	cfg := rConf.(*Config)

	scr := newKandjiScraper(params, cfg)

	s, err := scraper.NewMetrics(
		scr.scrape,
		scraper.WithStart(scr.start),
		scraper.WithShutdown(scr.shutdown),
	)
	if err != nil {
		return nil, err
	}

	return scraperhelper.NewMetricsController(
		&cfg.ControllerConfig,
		params,
		cons,
		scraperhelper.AddScraper(metadata.Type, s),
	)
}

func createLogsReceiver(
	_ context.Context,
	params receiver.Settings,
	rConf component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	cfg := rConf.(*Config)
	return newKandjiLogs(cfg, params, consumer), nil
}
