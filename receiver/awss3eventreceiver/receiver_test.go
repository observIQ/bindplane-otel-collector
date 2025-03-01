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

package awss3eventreceiver_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"

	rcvr "github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/bps3"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/bpsqs"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/fake"
)

func TestNewS3EventReceiver(t *testing.T) {
	set := receivertest.NewNopSettings()
	f := rcvr.NewFactory()
	cfg := f.CreateDefaultConfig().(*rcvr.Config)
	cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
	next := consumertest.NewNop()

	receiver, err := f.CreateLogs(context.Background(), set, cfg, next)
	require.NoError(t, err)
	assert.NotNil(t, receiver)
}

func TestNewS3EventReceiverValidationError(t *testing.T) {
	set := receivertest.NewNopSettings()
	f := rcvr.NewFactory()
	cfg := f.CreateDefaultConfig().(*rcvr.Config)
	cfg.SQSQueueURL = "https://invalid-url"
	next := consumertest.NewNop()

	receiver, err := f.CreateLogs(context.Background(), set, cfg, next)
	require.Error(t, err)
	assert.Nil(t, receiver)
	assert.Contains(t, err.Error(), "invalid SQS URL format")
}

// Test to verify region extraction works
func TestRegionExtractionFromSQSURL(t *testing.T) {
	set := receivertest.NewNopSettings()
	f := rcvr.NewFactory()
	cfg := f.CreateDefaultConfig().(*rcvr.Config)
	cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
	next := consumertest.NewNop()

	receiver, err := f.CreateLogs(context.Background(), set, cfg, next)
	require.NoError(t, err)
	assert.NotNil(t, receiver)

	// Verify the region was extracted correctly
	region, err := cfg.GetRegion()
	assert.NoError(t, err)
	assert.Equal(t, "us-west-2", region)
}

// TestStartShutdownUsingFakes tests the start and shutdown functions
func TestStartShutdown(t *testing.T) {
	var bpsqsNewClient func(aws.Config) bpsqs.Client
	var bps3NewClient func(aws.Config) bps3.Client
	bpsqsNewClient, bpsqs.NewClient = bpsqs.NewClient, fake.NewSQSClient([]types.Message{})
	bps3NewClient, bps3.NewClient = bps3.NewClient, fake.NewS3Client
	defer func() {
		bpsqs.NewClient = bpsqsNewClient
		bps3.NewClient = bps3NewClient
	}()

	ctx := context.Background()
	set := receivertest.NewNopSettings()
	f := rcvr.NewFactory()
	cfg := f.CreateDefaultConfig().(*rcvr.Config)
	cfg.SQSQueueURL = "https://sqs.us-west-2.amazonaws.com/123456789012/test-queue"
	next := consumertest.NewNop()

	receiver, err := f.CreateLogs(context.Background(), set, cfg, next)
	require.NoError(t, err)

	host := componenttest.NewNopHost()
	require.NoError(t, receiver.Start(ctx, host))
	require.NoError(t, receiver.Shutdown(ctx))
}
