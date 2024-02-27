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

package azureblobexporter // import "github.com/observiq/bindplane-agent/exporter/azureblobexporter"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

// partitionType is the type of partition to store blobs under
type partitionType string

const (
	minutePartition partitionType = "minute"
	hourPartition   partitionType = "hour"
)

// compressionType is the type of compression to apply to blobs
type compressionType string

const (
	noCompression   compressionType = "none"
	gzipCompression compressionType = "gzip"
)

// Config the configuration for the azureblob exporter
type Config struct {
	exporterhelper.TimeoutSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	exporterhelper.QueueSettings   `mapstructure:"sending_queue"`
	configretry.BackOffConfig      `mapstructure:"retry_on_failure"`

	// ConnectionString is the Azure Blob Storage connection key,
	// which can be found in the Azure Blob Storage resource on the Azure Portal. (no default)
	ConnectionString string `mapstructure:"connection_string"`

	// Container is the name of the user created storage container. (no default)
	Container string `mapstructure:"container"`

	// BlobPrefix is the blob prefix defined by the user. (no default)
	BlobPrefix string `mapstructure:"blob_prefix"`

	// RootFolder is the name of the root folder in path.
	RootFolder string `mapstructure:"root_folder"`

	// Partition is the time granularity of the blob.
	// Valid values are "hour" or "minute". Default: minute
	Partition partitionType `mapstructure:"partition"`

	// Compression is the type of compression to use.
	// Valid values are "none" or "gzip". Default: none
	Compression compressionType `mapstructure:"compression"`
}

// Validate validates the config.
func (c *Config) Validate() error {
	if c.ConnectionString == "" {
		return errors.New("connection_string is required")
	}

	if c.Container == "" {
		return errors.New("container is required")
	}

	switch c.Partition {
	case minutePartition, hourPartition:
	// do nothing
	default:
		return fmt.Errorf("unsupported partition type '%s'", c.Partition)
	}

	switch c.Compression {
	case noCompression, gzipCompression:
		return nil
	default:
		return fmt.Errorf("unsupported compression type: %s", c.Compression)
	}
}
