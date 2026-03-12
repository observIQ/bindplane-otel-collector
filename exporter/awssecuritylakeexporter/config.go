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

package awssecuritylakeexporter // import "github.com/observiq/bindplane-otel-collector/exporter/awssecuritylakeexporter"

import (
	"errors"

	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// Config is the configuration for the AWS Security Lake exporter.
type Config struct {
	TimeoutConfig    exporterhelper.TimeoutConfig                             `mapstructure:",squash"`
	QueueBatchConfig configoptional.Optional[exporterhelper.QueueBatchConfig] `mapstructure:"sending_queue"`
	BackOffConfig    configretry.BackOffConfig                                `mapstructure:"retry_on_failure"`
	TLS              configtls.ClientConfig                                   `mapstructure:"tls"`

	// Region is the AWS region where the Security Lake S3 bucket resides. (required)
	Region string `mapstructure:"region"`

	// S3Bucket is the name of the Security Lake S3 bucket. (required)
	S3Bucket string `mapstructure:"s3_bucket"`

	// S3Prefix is the S3 key prefix. Defaults to "ext/".
	S3Prefix string `mapstructure:"s3_prefix"`

	// SourceName is the custom source name registered in Security Lake. (required)
	SourceName string `mapstructure:"source_name"`

	// AccountID is the AWS account ID used in the partition path. (required)
	AccountID string `mapstructure:"account_id"`

	// RoleARN is an optional IAM role ARN to assume for S3 writes.
	RoleARN string `mapstructure:"role_arn"`

	// Endpoint is the optional custom endpoint to use for S3 writes.
	Endpoint string `mapstructure:"endpoint"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if err := c.TimeoutConfig.Validate(); err != nil {
		return err
	}
	if err := c.BackOffConfig.Validate(); err != nil {
		return err
	}
	if err := c.QueueBatchConfig.Validate(); err != nil {
		return err
	}
	if err := c.TLS.Validate(); err != nil {
		return err
	}
	if c.Region == "" {
		return errors.New("region is required")
	}
	if c.S3Bucket == "" {
		return errors.New("s3_bucket is required")
	}
	if c.SourceName == "" {
		return errors.New("source_name is required")
	}
	if c.AccountID == "" {
		return errors.New("account_id is required")
	}

	return nil
}
