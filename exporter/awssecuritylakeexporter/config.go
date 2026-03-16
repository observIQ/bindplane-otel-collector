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
	"fmt"
	"slices"

	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// OCSFVersion is the OCSF version to use for the logs.
type OCSFVersion string

const (
	// OCSFVersion1_0_0 is the OCSF version 1.0.0.
	OCSFVersion1_0_0 OCSFVersion = "1.0.0"
	// OCSFVersion1_1_0 is the OCSF version 1.1.0.
	OCSFVersion1_1_0 OCSFVersion = "1.1.0"
	// OCSFVersion1_2_0 is the OCSF version 1.2.0.
	OCSFVersion1_2_0 OCSFVersion = "1.2.0"
	// OCSFVersion1_3_0 is the OCSF version 1.3.0.
	OCSFVersion1_3_0 OCSFVersion = "1.3.0"
)

var ocsfVersions = []OCSFVersion{
	OCSFVersion1_0_0,
	OCSFVersion1_1_0,
	OCSFVersion1_2_0,
	OCSFVersion1_3_0,
}

// SecurityLakeCustomSource is the configuration for a custom source in Security Lake.
type SecurityLakeCustomSource struct {
	Name    string `mapstructure:"name"`
	ClassID int    `mapstructure:"class_id"`
}

// Config is the configuration for the AWS Security Lake exporter.
type Config struct {
	TimeoutConfig    exporterhelper.TimeoutConfig                             `mapstructure:",squash"`
	QueueBatchConfig configoptional.Optional[exporterhelper.QueueBatchConfig] `mapstructure:"sending_queue"`
	BackOffConfig    configretry.BackOffConfig                                `mapstructure:"retry_on_failure"`

	// OCSFVersion is the OCSF version to use for the logs.
	OCSFVersion OCSFVersion `mapstructure:"ocsf_version"`

	// Region is the AWS region where the Security Lake S3 bucket resides. (required)
	Region string `mapstructure:"region"`

	// S3Bucket is the name of the Security Lake S3 bucket. (required)
	S3Bucket string `mapstructure:"s3_bucket"`

	// CustomSources is the custom source name registered in Security Lake. (required)
	CustomSources []SecurityLakeCustomSource `mapstructure:"custom_sources"`

	// AccountID is the AWS account ID used in the partition path. (required)
	AccountID string `mapstructure:"account_id"`

	// RoleARN is an optional IAM role ARN to assume for S3 writes.
	RoleARN string `mapstructure:"role_arn"`

	// Endpoint is the optional custom endpoint to use for S3 writes.
	Endpoint string `mapstructure:"endpoint"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Region == "" {
		return errors.New("region is required")
	}
	if c.S3Bucket == "" {
		return errors.New("s3_bucket is required")
	}
	if len(c.CustomSources) == 0 {
		return errors.New("at least one custom_source is required")
	}
	for _, source := range c.CustomSources {
		if source.Name == "" {
			return errors.New("custom_source.name is required")
		}
		if source.ClassID == 0 {
			return errors.New("custom_source.class_id is required")
		}
	}
	if c.AccountID == "" {
		return errors.New("account_id is required")
	}
	if c.OCSFVersion == "" {
		return errors.New("ocsf_version is required")
	}
	if !slices.Contains(ocsfVersions, c.OCSFVersion) || getSchemaMap(c.OCSFVersion) == nil {
		return fmt.Errorf("invalid ocsf_version: %s", c.OCSFVersion)
	}
	return nil
}
