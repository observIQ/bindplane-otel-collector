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

package azureblob

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewAzureBlobClient(t *testing.T) {
	tests := []struct {
		name          string
		connectionStr string
		batchSize     int
		pageSize      int
		expectedError bool
	}{
		{
			name:          "Invalid connection string",
			connectionStr: "invalid",
			batchSize:     100,
			pageSize:      1000,
			expectedError: true,
		},
		{
			name:          "Valid connection string",
			connectionStr: "DefaultEndpointsProtocol=https;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1;",
			batchSize:     100,
			pageSize:      1000,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewAzureBlobClient(tt.connectionStr, tt.batchSize, tt.pageSize)
			if tt.expectedError {
				require.Error(t, err)
				require.Nil(t, client)
			} else {
				require.NoError(t, err)
				require.NotNil(t, client)
			}
		})
	}
}

func TestDownloadBlob(t *testing.T) {
	// Create a mock Azure client using testify/mock
	mockClient := &mockAzureClient{}

	client := &AzureClient{
		azClient:  mockClient,
		batchSize: 100,
		pageSize:  1000,
	}

	ctx := context.Background()
	container := "testcontainer"
	blobPath := "test/blob.txt"
	testData := []byte("test data content")
	buf := make([]byte, 1024)

	mockClient.On("DownloadBuffer", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		buf := args.Get(3).([]byte)
		copy(buf, testData)
	}).Return(int64(len(testData)), nil)

	t.Run("successful download", func(t *testing.T) {
		bytesDownloaded, err := client.DownloadBlob(ctx, container, blobPath, buf)
		require.NoError(t, err)
		require.Equal(t, int64(len(testData)), bytesDownloaded)
		require.Equal(t, string(testData), string(buf[:len(testData)]))
	})
}

func TestDeleteBlobSuccess(t *testing.T) {
	// Create a mock Azure client using testify/mock
	mockClient := &mockAzureClient{}
	client := &AzureClient{
		azClient:  mockClient,
		batchSize: 100,
		pageSize:  1000,
	}

	ctx := context.Background()
	container := "testcontainer"
	blobPath := "test/blob.txt"

	mockClient.On("DeleteBlob", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(azblob.DeleteBlobResponse{}, nil)
	err := client.DeleteBlob(ctx, container, blobPath)
	require.NoError(t, err)

}

func TestDeleteBlobFailure(t *testing.T) {
	// Create a mock Azure client using testify/mock
	mockClient := &mockAzureClient{}
	client := &AzureClient{
		azClient:  mockClient,
		batchSize: 100,
		pageSize:  1000,
	}

	ctx := context.Background()
	container := "testcontainer"
	blobPath := "test/blob.txt"

	mockClient.On("DeleteBlob", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(azblob.DeleteBlobResponse{}, errors.New("failed to delete"))
	err := client.DeleteBlob(ctx, container, blobPath)
	require.Error(t, err)
	require.Equal(t, "failed to delete", err.Error())
}

// mockAzureClient is a mock implementation of the Azure blob client
type mockAzureClient struct {
	mock.Mock
}

func (m *mockAzureClient) NewListBlobsFlatPager(containerName string, options *azblob.ListBlobsFlatOptions) *runtime.Pager[azblob.ListBlobsFlatResponse] {
	args := m.Called(containerName, options)
	return args.Get(0).(*runtime.Pager[azblob.ListBlobsFlatResponse])
}

func (m *mockAzureClient) DownloadBuffer(ctx context.Context, containerName string, blobPath string, buffer []byte, options *azblob.DownloadBufferOptions) (int64, error) {
	args := m.Called(ctx, containerName, blobPath, buffer, options)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockAzureClient) DeleteBlob(ctx context.Context, containerName string, blobPath string, options *azblob.DeleteBlobOptions) (azblob.DeleteBlobResponse, error) {
	args := m.Called(ctx, containerName, blobPath, options)
	return args.Get(0).(azblob.DeleteBlobResponse), args.Error(1)
}
