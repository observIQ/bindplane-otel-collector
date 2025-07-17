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

// Package constants provides shared constants for the AWS S3 Event receiver.
package constants

const (
	// NotificationTypeS3 represents direct S3 event notifications
	NotificationTypeS3 = "s3"
	// NotificationTypeSNS represents S3 events wrapped in SNS notifications
	NotificationTypeSNS = "sns"
)
