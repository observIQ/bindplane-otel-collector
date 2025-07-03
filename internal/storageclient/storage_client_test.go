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

package storageclient //import "github.com/observiq/bindplane-otel-collector/internal/storageclient"

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/storage/filestorage"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/pipeline"

	"github.com/stretchr/testify/require"
)

type TestStorageData struct {
	Data string
}

func (t *TestStorageData) Marshal() ([]byte, error) {
	return json.Marshal(t)
}

func (t *TestStorageData) Unmarshal(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	return json.Unmarshal(data, t)
}

func TestNopStorage(t *testing.T) {
	storage := NewNopStorage()

	require.NoError(t, storage.SaveStorageData(context.Background(), "key", &TestStorageData{}))

	checkpoint := &TestStorageData{}
	require.NoError(t, storage.LoadStorageData(context.Background(), "key", checkpoint))
	require.Equal(t, &TestStorageData{}, checkpoint)
}

func TestExtensionStorage(t *testing.T) {
	// create a file storage extension
	factory := filestorage.NewFactory()
	cfg := factory.CreateDefaultConfig().(*filestorage.Config)
	cfg.Directory = t.TempDir()

	id := component.NewIDWithName(component.MustNewType("file_storage"), "file_extension")
	extension, err := factory.Create(context.Background(), extension.Settings{ID: id}, cfg)
	require.NoError(t, err)

	// start the extension
	extension.Start(context.Background(), &testHost{})
	defer extension.Shutdown(context.Background())

	// create a host with the extension
	host := &testHost{
		components: map[component.ID]component.Component{
			id: extension,
		},
	}

	// create a storage client with the extension
	thisID := component.NewIDWithName(component.MustNewType("storage_client"), "test")
	storage, err := NewStorageClient(context.Background(), host, id, thisID, pipeline.SignalLogs)
	require.NoError(t, err)

	// save data
	storage.SaveStorageData(context.Background(), "key", &TestStorageData{Data: "test"})

	// load data
	checkpoint := &TestStorageData{}
	require.NoError(t, storage.LoadStorageData(context.Background(), "key", checkpoint))
	require.Equal(t, &TestStorageData{Data: "test"}, checkpoint)

	// load empty data
	checkpoint = &TestStorageData{}
	require.NoError(t, storage.LoadStorageData(context.Background(), "key2", checkpoint))
	require.Equal(t, &TestStorageData{}, checkpoint)

	// close the storage client
	require.NoError(t, storage.Close(context.Background()))
}

type testHost struct {
	components map[component.ID]component.Component
}

func (t *testHost) GetExtensions() map[component.ID]component.Component {
	return t.components
}
