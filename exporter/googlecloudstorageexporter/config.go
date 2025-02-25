package googlecloudstorageexporter // import "github.com/observiq/bindplane-otel-collector/exporter/googlecloudstorageexporter"

import "errors"

type Config struct {
	ProjectID  string `mapstructure:"project_id"`
	BucketName string `mapstructure:"bucket_name"`
	Location   string `mapstructure:"location"`
	StorageClass string `mapstructure:"storage_class"`
	FolderName string `mapstructure:"folder_name"`
	BlobPrefix string `mapstructure:"blob_prefix"`
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
