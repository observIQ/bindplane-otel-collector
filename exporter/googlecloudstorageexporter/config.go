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

	// Validate credentials - only one form of credentials should be provided
	if c.Credentials != "" && c.CredentialsFile != "" {
		return errors.New("only one form of credentials should be provided")
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
