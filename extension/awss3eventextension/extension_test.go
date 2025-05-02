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

package awss3eventextension

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension/extensiontest"

	"github.com/observiq/bindplane-otel-collector/extension/awss3eventextension/internal/metadata"
	"github.com/observiq/bindplane-otel-collector/internal/aws/client"
	"github.com/observiq/bindplane-otel-collector/internal/aws/fake"
)

func TestExtensionLifecycle(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.SQSQueueURL = "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue"
	cfg.Directory = "/tmp/s3event"

	ext, err := factory.Create(context.Background(), extensiontest.NewNopSettings(metadata.Type), cfg)
	require.NoError(t, err)
	require.NotNil(t, ext)

	ctx := context.Background()
	host := componenttest.NewNopHost()

	ext.Start(ctx, host)
	require.NoError(t, err)

	ext.Shutdown(ctx)
	require.NoError(t, err)
}

func TestExtensionS3Events(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	fakeAWS := client.NewClient(aws.Config{}).(*fake.AWS)
	objectSet := map[string]map[string]string{
		"test-bucket-1": {
			"test-key-a": "test-body-a",
			"test-key-b": "test-body-b",
		},
		"test-bucket-2": {
			"test-key-c": "test-body-c",
			"test-key-d": "test-body-d",
		},
	}
	fakeAWS.CreateObjects(t, objectSet)

	tmpDir := t.TempDir()

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.SQSQueueURL = "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue"
	cfg.Directory = tmpDir
	cfg.EventFormat = "aws_s3"
	cfg.StandardPollInterval = 50 * time.Millisecond

	ext, err := factory.Create(context.Background(), extensiontest.NewNopSettings(metadata.Type), cfg)
	require.NoError(t, err)
	require.NotNil(t, ext)

	require.NoError(t, ext.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, ext.Shutdown(context.Background()))
	}()

	require.Eventually(t, func() bool {
		dirs, err := os.ReadDir(tmpDir)
		require.NoError(t, err)
		if len(dirs) != 2 {
			return false
		}
		for _, dir := range dirs {
			if !dir.IsDir() {
				return false
			}
			bucket := dir.Name()
			objects, err := os.ReadDir(filepath.Join(tmpDir, bucket))
			require.NoError(t, err)
			if len(objects) != 2 {
				return false
			}
			for _, object := range objects {
				body, err := os.ReadFile(filepath.Join(tmpDir, bucket, object.Name()))
				require.NoError(t, err)
				if string(body) != objectSet[bucket][object.Name()] {
					return false
				}
			}
		}
		return true
	}, time.Second*2, time.Millisecond*100)
}

func TestExtensionFDREvents(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t, fake.WithFDRBodyMarshaler())()

	fakeAWS := client.NewClient(aws.Config{}).(*fake.AWS)
	objectSet := map[string]map[string]string{
		"test-bucket-1": {
			"test-key-a": "test-body-a",
			"test-key-b": "test-body-b",
		},
		"test-bucket-2": {
			"test-key-c": "test-body-c",
			"test-key-d": "test-body-d",
		},
	}
	fakeAWS.CreateObjects(t, objectSet)

	tmpDir := t.TempDir()

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.SQSQueueURL = "https://sqs.us-east-1.amazonaws.com/123456789012/test-queue"
	cfg.Directory = tmpDir
	cfg.EventFormat = "crowdstrike_fdr"
	cfg.StandardPollInterval = 50 * time.Millisecond

	ext, err := factory.Create(context.Background(), extensiontest.NewNopSettings(metadata.Type), cfg)
	require.NoError(t, err)
	require.NotNil(t, ext)

	require.NoError(t, ext.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, ext.Shutdown(context.Background()))
	}()

	require.Eventually(t, func() bool {
		dirs, err := os.ReadDir(tmpDir)
		require.NoError(t, err)
		if len(dirs) != 2 {
			return false
		}
		for _, dir := range dirs {
			if !dir.IsDir() {
				return false
			}
			bucket := dir.Name()
			objects, err := os.ReadDir(filepath.Join(tmpDir, bucket))
			require.NoError(t, err)
			if len(objects) != 2 {
				return false
			}
			for _, object := range objects {
				body, err := os.ReadFile(filepath.Join(tmpDir, bucket, object.Name()))
				require.NoError(t, err)
				if string(body) != objectSet[bucket][object.Name()] {
					return false
				}
			}
		}
		return true
	}, time.Second*2, time.Millisecond*100)
}
