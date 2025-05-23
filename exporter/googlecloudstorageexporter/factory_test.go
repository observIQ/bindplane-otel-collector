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
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

func Test_createDefaultConfig(t *testing.T) {
	expectedCfg := &Config{
		ProjectID:          "",
		BucketName:         "",
		BucketLocation:     "",
		BucketStorageClass: "",
		FolderName:         "",
		ObjectPrefix:       "",
		Credentials:        "",
		CredentialsFile:    "",
		Partition:          minutePartition,
		Compression:        noCompression,
		TimeoutConfig:      exporterhelper.NewDefaultTimeoutConfig(),
		QueueConfig:        exporterhelper.NewDefaultQueueConfig(),
		BackOffConfig:      configretry.NewDefaultBackOffConfig(),
	}

	actual := createDefaultConfig()
	require.Equal(t, expectedCfg, actual)
}
