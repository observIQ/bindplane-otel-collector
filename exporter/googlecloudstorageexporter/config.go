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

// partitionType is the type of partition to store objects under
type partitionType string

const (
	minutePartition partitionType = "minute"
	hourPartition   partitionType = "hour"
)

// compressionType is the type of compression to apply to objects
type compressionType string

const (
	noCompression   compressionType = "none"
	gzipCompression compressionType = "gzip"
)

// Config is the configuration for the googlecloudstorage exporter
type Config struct {
	// ProjectID is the ID of the Google Cloud project the bucket belongs to.
	ProjectID string `mapstructure:"project_id"`

	// BucketName is the name of the bucket to store objects in.
	BucketName string `mapstructure:"bucket_name"`

	// BucketLocation is the location of the bucket.
	BucketLocation string `mapstructure:"bucket_location"`

	// BucketStorageClass is the storage class of the bucket.
	BucketStorageClass string `mapstructure:"bucket_storage_class"`

	// FolderName is the name of the folder to store objects under.
	FolderName string `mapstructure:"folder_name"`

	// ObjectPrefix is the prefix to add to the object name.
	ObjectPrefix string `mapstructure:"object_prefix"`

	// Credentials and CredentialsFile are mutually exclusive and provide authentication to Google Cloud Storage.
	Credentials     string `mapstructure:"credentials"`
	CredentialsFile string `mapstructure:"credentials_file"`

	// Partition is the time granularity of the object.
	// Valid values are "hour" or "minute". Default: minute
	Partition partitionType `mapstructure:"partition"`

	// Compression is the type of compression to use.
	// Valid values are "none" or "gzip". Default: none
	Compression compressionType `mapstructure:"compression"`

	exporterhelper.TimeoutConfig `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	exporterhelper.QueueConfig   `mapstructure:"sending_queue"`
	configretry.BackOffConfig    `mapstructure:"retry_on_failure"`
}

// Validate validates the config.
func (c *Config) Validate() error {
	if c.BucketName == "" {
		return errors.New("bucket_name is required")
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
