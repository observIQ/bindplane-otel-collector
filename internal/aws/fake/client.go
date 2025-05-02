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

// Package fake provides fake implementations of AWS clients for testing
package fake

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/observiq/bindplane-otel-collector/internal/aws/client"
	"github.com/observiq/bindplane-otel-collector/internal/aws/event"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.uber.org/zap"
)

var _ client.Client = &AWS{}

// AWS is a fake AWS client
type AWS struct {
	set component.TelemetrySettings

	s3Client  *s3Client
	sqsClient *sqsClient

	eventMarshaler event.Marshaler
}

// ClientOption is a function that modifies the AWS client
type ClientOption func(*AWS)

// WithFDRBodyMarshaler sets the event marshaler to the FDR marshaler
func WithFDRBodyMarshaler() ClientOption {
	return func(f *AWS) {
		f.eventMarshaler = event.NewFDRMarshaler(f.set)
	}
}

// SetFakeConstructorForTest sets the fake constructor for the AWS client
// It returns a function that restores the original constructor
// It is intended to be used in a defer statement
// e.g. defer fake.SetFakeConstructorForTest(t)()
func SetFakeConstructorForTest(t *testing.T, opts ...ClientOption) func() {
	realNewClient := client.NewClient
	set := componenttest.NewNopTelemetrySettings()
	set.Logger = zap.NewNop()
	client.NewClient = func(_ aws.Config) client.Client {
		f := &AWS{
			set:            set,
			s3Client:       NewS3Client(t).(*s3Client),
			sqsClient:      NewSQSClient(t).(*sqsClient),
			eventMarshaler: event.NewS3Marshaler(set),
		}
		for _, opt := range opts {
			opt(f)
		}
		return f
	}

	// Caller should defer this function to restore the original constructor
	return func() { client.NewClient = realNewClient }
}

// S3 returns the fake S3 client
func (a *AWS) S3() client.S3Client {
	return a.s3Client
}

// SQS returns the fake SQS client
func (a *AWS) SQS() client.SQSClient {
	return a.sqsClient
}

// CreateObjects creates objects in the fake S3 client and adds a corresponding message to the fake SQS client
func (a *AWS) CreateObjects(t *testing.T, objects map[string]map[string]string) {
	a.CreateObjectsWithEventType(t, "s3:ObjectCreated:Put", objects)
}

// CreateObjectsWithEventType creates objects in the fake S3 client and adds a corresponding message
// to the fake SQS client with the specified event type
func (a *AWS) CreateObjectsWithEventType(t *testing.T, eventType string, objects map[string]map[string]string) {
	objectCreated := make([]event.S3Object, 0, len(objects))
	for bucket, keys := range objects {
		for key, body := range keys {
			a.s3Client.putObject(bucket, key, body)
			objectCreated = append(objectCreated, event.S3Object{
				EventType: eventType,
				Bucket:    bucket,
				Key:       key,
				Size:      int64(len(body)),
			})
		}
	}
	a.notifySQS(t, objectCreated)
}

func (a *AWS) notifySQS(t *testing.T, objectsCreated []event.S3Object) {
	body, err := a.eventMarshaler.Marshal(objectsCreated)
	require.NoError(t, err)
	for _, body := range body {
		a.sqsClient.sendMessage(body)
	}
}
