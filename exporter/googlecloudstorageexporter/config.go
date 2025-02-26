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

package googlecloudstorageexporter // import "github.com/observiq/bindplane-otel-collector/exporter/googlecloudstorageexporter"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

type partitionType string

const (
	minutePartition partitionType = "minute"
	hourPartition   partitionType = "hour"
)

type compressionType string

const (
	noCompression   compressionType = "none"
	gzipCompression compressionType = "gzip"
)

type Config struct {
	ProjectID  string `mapstructure:"project_id"`
	BucketName string `mapstructure:"bucket_name"`
	Location   string `mapstructure:"location"`
	StorageClass string `mapstructure:"storage_class"`
	FolderName string `mapstructure:"folder_name"`
	ObjectPrefix string `mapstructure:"object_prefix"`

	Credentials string `mapstructure:"credentials"`
	CredentialsFile string `mapstructure:"credentials_file"`

	Partition partitionType `mapstructure:"partition"`
	Compression compressionType `mapstructure:"compression"`

	exporterhelper.TimeoutConfig `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	exporterhelper.QueueConfig   `mapstructure:"sending_queue"`
	configretry.BackOffConfig `mapstructure:"retry_on_failure"`
}

func (c *Config) Validate() error {
	if c.BucketName == "" {
		return errors.New("bucket_name is required")
	}
	if c.ProjectID == "" {
		return errors.New("project_id is required")
	}
	if c.Location == "" {
		return errors.New("location is required")
	}

	// Validate credentials - both can be empty (for default credentials) but both cannot be set
	if c.Credentials != "" && c.CredentialsFile != "" {
		return errors.New("cannot specify both credentials and credentials_file")
	}

	switch c.Partition {
	case minutePartition, hourPartition:
	// do nothing
	default:
		return fmt.Errorf("unsupported partition type '%s'", c.Partition)
	}

	switch c.Compression {
	case noCompression, gzipCompression:
	// do nothing	
	default:
		return fmt.Errorf("unsupported compression type: %s", c.Compression)
	}
	
	return nil
}
