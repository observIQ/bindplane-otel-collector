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

package googlecloudexporter

import (
	"os"

	"github.com/GoogleCloudPlatform/opentelemetry-operations-collector/processor/normalizesumsprocessor"
	"github.com/mitchellh/mapstructure"
	"github.com/observiq/observiq-otel-collector/processor/resourceattributetransposerprocessor"
	gcp "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudexporter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourceprocessor"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.uber.org/multierr"
)

const (
	defaultMetricPrefix = "workloads.googleapis.com"
	defaultUserAgent    = "observIQ-otel-collector"
	defaultLocation     = "global"
	genericNodeResource = "generic_node"
)

// Config is the config the google cloud exporter
type Config struct {
	config.ExporterSettings `mapstructure:",squash"`
	Location                string                                       `mapstructure:"location"`
	Namespace               string                                       `mapstructure:"namespace"`
	GCPConfig               *gcp.Config                                  `mapstructure:",squash"`
	BatchConfig             *batchprocessor.Config                       `mapstructure:"batch"`
	NormalizeConfig         *normalizesumsprocessor.Config               `mapstructure:"normalize"`
	DetectorConfig          *resourcedetectionprocessor.Config           `mapstructure:"detector"`
	AttributerConfig        *resourceprocessor.Config                    `mapstructure:"attributer"`
	TransposerConfig        *resourceattributetransposerprocessor.Config `mapstructure:"transposer"`
}

// Validate validates the config
func (c *Config) Validate() error {
	var err error
	err = multierr.Append(err, c.GCPConfig.Validate())
	err = multierr.Append(err, c.BatchConfig.Validate())
	err = multierr.Append(err, c.NormalizeConfig.Validate())
	err = multierr.Append(err, c.DetectorConfig.Validate())
	err = multierr.Append(err, c.AttributerConfig.Validate())
	err = multierr.Append(err, c.TransposerConfig.Validate())
	return err
}

// createDefaultConfig creates the default config for the exporter
func createDefaultConfig() config.Exporter {
	defaultNamespace, _ := os.Hostname()
	return &Config{
		ExporterSettings: config.NewExporterSettings(config.NewComponentID(typeStr)),
		Location:         defaultLocation,
		Namespace:        defaultNamespace,
		GCPConfig:        createDefaultGCPConfig(),
		BatchConfig:      createDefaultBatchConfig(),
		NormalizeConfig:  createDefaultNormalizerConfig(),
		DetectorConfig:   createDefaultDetectorConfig(),
		AttributerConfig: createDefaultAttributerConfig(),
		TransposerConfig: createDefaultTransposerConfig(),
	}
}

// createDefaultGCPConfig creates a default GCP config
func createDefaultGCPConfig() *gcp.Config {
	gcpFactory := gcp.NewFactory()
	gcpConfig := gcpFactory.CreateDefaultConfig().(*gcp.Config)
	gcpConfig.RetrySettings.Enabled = false
	gcpConfig.UserAgent = defaultUserAgent
	gcpConfig.MetricConfig.Prefix = defaultMetricPrefix
	gcpConfig.ResourceMappings = []gcp.ResourceMapping{
		{
			TargetType: genericNodeResource,
			LabelMappings: []gcp.LabelMapping{
				{
					SourceKey: "host.name",
					TargetKey: "node_id",
				},
				{
					SourceKey: "location",
					TargetKey: "location",
				},
				{
					SourceKey: "namespace",
					TargetKey: "namespace",
				},
			},
		},
	}

	return gcpConfig
}

// createDefaultBatchConfig creates a default batch config
func createDefaultBatchConfig() *batchprocessor.Config {
	batchFactory := batchprocessor.NewFactory()
	batchConfig := batchFactory.CreateDefaultConfig().(*batchprocessor.Config)
	return batchConfig
}

// createDefaultNormalizerConfig creates a default normalizer config
func createDefaultNormalizerConfig() *normalizesumsprocessor.Config {
	normalizeFactory := normalizesumsprocessor.NewFactory()
	return normalizeFactory.CreateDefaultConfig().(*normalizesumsprocessor.Config)
}

// createDefaultDetectorConfig creates a default detector config
func createDefaultDetectorConfig() *resourcedetectionprocessor.Config {
	detectorFactory := resourcedetectionprocessor.NewFactory()
	detectorConfig := detectorFactory.CreateDefaultConfig().(*resourcedetectionprocessor.Config)
	detectorConfig.Detectors = []string{"system"}
	detectorConfig.DetectorConfig.SystemConfig.HostnameSources = []string{"os"}
	return detectorConfig
}

// createDefaultAttributerConfig creates a default attributer config
func createDefaultAttributerConfig() *resourceprocessor.Config {
	attributerFactory := resourceprocessor.NewFactory()
	attributerConfig := attributerFactory.CreateDefaultConfig().(*resourceprocessor.Config)
	return attributerConfig
}

// createDefaultTransposerConfig creates a default transposer config
func createDefaultTransposerConfig() *resourceattributetransposerprocessor.Config {
	transposerFactory := resourceattributetransposerprocessor.NewFactory()
	transposerConfig := transposerFactory.CreateDefaultConfig().(*resourceattributetransposerprocessor.Config)
	transposerConfig.Operations = []resourceattributetransposerprocessor.CopyResourceConfig{
		{
			From: "process.pid",
			To:   "pid",
		},
		{
			From: "process.executable.name",
			To:   "binary",
		},
	}
	return transposerConfig
}

// addGenericAttributes adds generic node attributes to the resource processor config
func addGenericAttributes(cfg resourceprocessor.Config, namespace, location string) *resourceprocessor.Config {
	defaultCfg := resourceprocessor.NewFactory().CreateDefaultConfig().(*resourceprocessor.Config)
	params := map[string]interface{}{
		"attributes": []map[string]interface{}{
			{
				"key":    "namespace",
				"value":  namespace,
				"action": "upsert",
			},
			{
				"key":    "location",
				"value":  location,
				"action": "upsert",
			},
		},
	}
	_ = mapstructure.Decode(&params, &defaultCfg)
	cfg.AttributesActions = append(cfg.AttributesActions, defaultCfg.AttributesActions...)
	return &cfg
}
