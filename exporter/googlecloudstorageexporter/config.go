package googlecloudstorageexporter // import "github.com/observiq/bindplane-otel-collector/exporter/googlecloudstorageexporter"

import (
	"errors"

	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

type Config struct {
	exporterhelper.TimeoutConfig `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	exporterhelper.QueueConfig   `mapstructure:"sending_queue"`
	configretry.BackOffConfig `mapstructure:"retry_on_failure"`

	ProjectID  string `mapstructure:"project_id"`
	BucketName string `mapstructure:"bucket_name"`
	Location   string `mapstructure:"location"`
	StorageClass string `mapstructure:"storage_class"`
	FolderName string `mapstructure:"folder_name"`
	ObjectPrefix string `mapstructure:"object_prefix"`
	// CredentialsFile string `mapstructure:"credentials_file"`
}

func (c *Config) Validate() error {
	if c.BucketName == "" {
		return errors.New("bucket_name is required")
	}
	if c.ProjectID == "" {
		return errors.New("project_id is required")
	}
	return nil

	// if c.FolderName == "" {
	// 	return errors.New("folder_name is required")
	// }

	// if c.BlobPrefix == "" {
	// 	return errors.New("blob_prefix is required")
	// }
	
	
}
