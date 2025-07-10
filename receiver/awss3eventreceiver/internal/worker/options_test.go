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

package worker_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"

	"github.com/observiq/bindplane-otel-collector/internal/aws/client"
	"github.com/observiq/bindplane-otel-collector/internal/aws/fake"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/metadata"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/worker"
)

func TestWithBucketNameFilter(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	set := componenttest.NewNopTelemetrySettings()
	sink := new(consumertest.LogsSink)
	fakeClient := client.NewClient(aws.Config{})

	filter := regexp.MustCompile("test-bucket")
	w := worker.New(
		set, sink, fakeClient, 1024, 100,
		time.Second, time.Minute, time.Hour,
		worker.WithBucketNameFilter(filter),
	)

	require.NotNil(t, w)

	t.Run("bucket filter applied correctly", func(t *testing.T) {
		require.True(t, filter.MatchString("test-bucket"))
		require.False(t, filter.MatchString("other-bucket"))
	})
}

func TestWithObjectKeyFilter(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	set := componenttest.NewNopTelemetrySettings()
	sink := new(consumertest.LogsSink)
	fakeClient := client.NewClient(aws.Config{})

	filter := regexp.MustCompile(`\.log$`)
	w := worker.New(
		set, sink, fakeClient, 1024, 100,
		time.Second, time.Minute, time.Hour,
		worker.WithObjectKeyFilter(filter),
	)

	require.NotNil(t, w)

	t.Run("object key filter applied correctly", func(t *testing.T) {
		require.True(t, filter.MatchString("app.log"))
		require.False(t, filter.MatchString("app.txt"))
	})
}

func TestWithTelemetryBuilder(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	set := componenttest.NewNopTelemetrySettings()
	sink := new(consumertest.LogsSink)
	fakeClient := client.NewClient(aws.Config{})

	tb, err := metadata.NewTelemetryBuilder(set)
	require.NoError(t, err)

	w := worker.New(
		set, sink, fakeClient, 1024, 100,
		time.Second, time.Minute, time.Hour,
		worker.WithTelemetryBuilder(tb),
	)

	require.NotNil(t, w)
}

func TestWithTelemetryBuilder_Nil(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	set := componenttest.NewNopTelemetrySettings()
	sink := new(consumertest.LogsSink)
	fakeClient := client.NewClient(aws.Config{})

	w := worker.New(
		set, sink, fakeClient, 1024, 100,
		time.Second, time.Minute, time.Hour,
		worker.WithTelemetryBuilder(nil),
	)

	require.NotNil(t, w)
}

func TestMultipleOptions(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	set := componenttest.NewNopTelemetrySettings()
	sink := new(consumertest.LogsSink)
	fakeClient := client.NewClient(aws.Config{})

	bucketFilter := regexp.MustCompile("prod-.*")
	keyFilter := regexp.MustCompile(`.*\.json$`)
	tb, err := metadata.NewTelemetryBuilder(set)
	require.NoError(t, err)

	w := worker.New(
		set, sink, fakeClient, 1024, 100,
		time.Second, time.Minute, time.Hour,
		worker.WithBucketNameFilter(bucketFilter),
		worker.WithObjectKeyFilter(keyFilter),
		worker.WithTelemetryBuilder(tb),
	)

	require.NotNil(t, w)

	t.Run("all filters applied correctly", func(t *testing.T) {
		require.True(t, bucketFilter.MatchString("prod-logs"))
		require.False(t, bucketFilter.MatchString("dev-logs"))
		require.True(t, keyFilter.MatchString("data.json"))
		require.False(t, keyFilter.MatchString("data.txt"))
	})
}

func TestNoOptions(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	set := componenttest.NewNopTelemetrySettings()
	sink := new(consumertest.LogsSink)
	fakeClient := client.NewClient(aws.Config{})

	w := worker.New(
		set, sink, fakeClient, 1024, 100,
		time.Second, time.Minute, time.Hour,
	)

	require.NotNil(t, w)
}

func TestWithNilFilters(t *testing.T) {
	defer fake.SetFakeConstructorForTest(t)()

	set := componenttest.NewNopTelemetrySettings()
	sink := new(consumertest.LogsSink)
	fakeClient := client.NewClient(aws.Config{})

	w := worker.New(
		set, sink, fakeClient, 1024, 100,
		time.Second, time.Minute, time.Hour,
		worker.WithBucketNameFilter(nil),
		worker.WithObjectKeyFilter(nil),
	)

	require.NotNil(t, w)
}
