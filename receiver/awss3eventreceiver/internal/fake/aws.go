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
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/bpaws"
	"github.com/stretchr/testify/require"
)

var _ bpaws.Client = &AWS{}

type AWS struct {
	s3Client  *s3Client
	sqsClient *sqsClient

	count   int
	countMu sync.Mutex
}

// SetFakeConstructorForTest sets the fake constructor for the AWS client
// It returns a function that restores the original constructor
// It is intended to be used in a defer statement
// e.g. defer fake.SetFakeConstructorForTest()()
func SetFakeConstructorForTest() func() {
	realNewClient := bpaws.NewClient
	bpaws.NewClient = NewClient
	return func() {
		bpaws.NewClient = realNewClient
	}
}

func NewClient(cfg aws.Config) bpaws.Client {
	return &AWS{
		s3Client:  NewS3Client(cfg).(*s3Client),
		sqsClient: NewSQSClient(cfg).(*sqsClient),
	}
}

func (a *AWS) S3() bpaws.S3Client {
	return a.s3Client
}

func (a *AWS) SQS() bpaws.SQSClient {
	return a.sqsClient
}

// CreateObjects creates objects in the fake S3 client and adds a corresponding message to the fake SQS client
func (a *AWS) CreateObjects(t *testing.T, objects map[string]map[string]string) {
	records := make([]events.S3EventRecord, 0, len(objects))
	for bucket, keys := range objects {
		for key, body := range keys {
			a.s3Client.putObject(bucket, key, body)
			records = append(records, newS3Record(bucket, key, body))
		}
	}
	msg := a.newS3Event(t, records...)
	a.sqsClient.sendMessage(msg)
}

func (a *AWS) newS3Event(t *testing.T, records ...events.S3EventRecord) types.Message {
	a.countMu.Lock()
	receiptHandle := aws.String(fmt.Sprintf("receiptHandle-%d", a.count))
	a.count++
	a.countMu.Unlock()

	body, err := json.Marshal(events.S3Event{Records: records})
	require.NoError(t, err)

	return types.Message{
		MessageId:     aws.String(fmt.Sprintf("messageId-%d", a.count)),
		Body:          aws.String(string(body)),
		ReceiptHandle: receiptHandle,
	}
}

func newS3Record(bucket string, key string, body string) events.S3EventRecord {
	return events.S3EventRecord{
		EventName:   "ObjectCreated:Put",
		EventSource: "aws:s3",
		EventTime:   time.Now(),
		S3: events.S3Entity{
			Bucket: events.S3Bucket{
				Name: bucket,
			},
			Object: events.S3Object{
				Key:  key,
				Size: int64(len(body)),
			},
		},
	}
}
