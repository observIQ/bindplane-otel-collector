// Copyright  observIQ, Inc.
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

// Package models contain request and response structures used for the SAPControl Web Service Interface
package models // import "github.com/observiq/observiq-otel-collector/receiver/sapnetweaverreceiver/internal/models"

import "encoding/xml"

// GetInstanceProperties is an xml request struct used to return a response struct
type GetInstanceProperties struct {
	XMLName xml.Name `xml:"urn:SAPControl GetInstanceProperties"`
}

// GetAlertTree is an xml request struct used to return a response struct
type GetAlertTree struct {
	XMLName xml.Name `xml:"urn:SAPControl GetAlertTree"`
}

// EnqGetStatistic is an xml request struct used to return a response struct
type EnqGetStatistic struct {
	XMLName xml.Name `xml:"urn:SAPControl EnqGetStatistic"`
}

// GetSystemInstanceList is an xml request struct used to return a response struct
type GetSystemInstanceList struct {
	XMLName xml.Name `xml:"urn:SAPControl GetSystemInstanceList"`
	Timeout int32    `xml:"timeout" json:"timeout"`
}

// GetProcessList is an xml request struct used to return a response struct
type GetProcessList struct {
	XMLName xml.Name `xml:"urn:SAPControl GetProcessList"`
}

// GetQueueStatistic is an xml request struct used to return a response struct
type GetQueueStatistic struct {
	XMLName xml.Name `xml:"urn:SAPControl GetQueueStatistic"`
}

// ABAPGetSystemWPTable is an xml request struct used to return a response struct
type ABAPGetSystemWPTable struct {
	XMLName    xml.Name `xml:"urn:SAPControl ABAPGetSystemWPTable"`
	Activeonly bool     `xml:"activeonly" json:"activeonly"`
}
