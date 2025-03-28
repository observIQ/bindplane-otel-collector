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

// Package bpaws provides a client for AWS services.
package bpaws // import "github.com/observiq/bindplane-otel-collector/receiver/awss3eventreceiver/internal/bpaws"

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// NewClient creates a new AWS client
var NewClient = func(cfg aws.Config) Client {
	return &client{
		s3Client:  s3.NewFromConfig(cfg),
		sqsClient: sqs.NewFromConfig(cfg),
	}
}

// Client is a client for AWS services
type Client interface {
	S3() S3Client
	SQS() SQSClient
}

type client struct {
	s3Client  S3Client
	sqsClient SQSClient
}

func (c *client) S3() S3Client {
	return c.s3Client
}

func (c *client) SQS() SQSClient {
	return c.sqsClient
}
