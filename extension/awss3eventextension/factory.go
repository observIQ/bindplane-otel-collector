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

package awss3eventextension // import "github.com/observiq/bindplane-otel-collector/extension/awss3eventextension"

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/google/uuid"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"

	"github.com/observiq/bindplane-otel-collector/extension/awss3eventextension/internal/metadata"
	"github.com/observiq/bindplane-otel-collector/extension/awss3eventextension/internal/worker"
	"github.com/observiq/bindplane-otel-collector/internal/aws/client"
	"github.com/observiq/bindplane-otel-collector/internal/aws/event"
)

// NewFactory creates a new extension factory
func NewFactory() extension.Factory {
	return extension.NewFactory(
		metadata.Type,
		createDefaultConfig,
		createExtension,
		metadata.ExtensionStability,
	)
}

// createDefaultConfig creates a default configuration
func createDefaultConfig() component.Config {
	return &Config{
		StandardPollInterval: 15 * time.Second,
		MaxPollInterval:      120 * time.Second,
		PollingBackoffFactor: 2,
		VisibilityTimeout:    300 * time.Second,
		Workers:              5,
		EventFormat:          eventFormatAWSS3,
	}
}

func createExtension(_ context.Context, set extension.Settings, cfg component.Config) (extension.Extension, error) {
	extCfg := cfg.(*Config)

	if extCfg.SQSQueueURL == "" {
		return nil, fmt.Errorf("'sqs_queue_url' is required")
	}

	region, err := client.ParseRegionFromSQSURL(extCfg.SQSQueueURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract region from SQS URL: %w", err)
	}

	awsConfig, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
	}

	// Clean path to ensure no file injection
	extCfg.Directory = filepath.Clean(extCfg.Directory)

	// Create and remove a subdirectory to validate permissions
	dir := filepath.Join(extCfg.Directory, uuid.New().String())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("insufficient permissions to create directory: %w", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return nil, fmt.Errorf("insufficient permissions to remove directory: %w", err)
	}

	var unmarshaler event.Unmarshaler
	switch extCfg.EventFormat {
	case eventFormatAWSS3:
		unmarshaler = event.NewS3Unmarshaler(set.TelemetrySettings)
	case eventFormatCrowdstrikeFDR:
		unmarshaler = event.NewFDRUnmarshaler(set.TelemetrySettings)
	default:
		return nil, fmt.Errorf("unsupported event format: %s", extCfg.EventFormat)
	}

	return &awsS3EventExtension{
		cfg:       extCfg,
		telemetry: set.TelemetrySettings,
		sqsClient: client.NewClient(awsConfig).SQS(),
		workerPool: sync.Pool{
			New: func() any {
				return worker.New(set.TelemetrySettings, awsConfig, unmarshaler, extCfg.Directory)
			},
		},
	}, nil
}
