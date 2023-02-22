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

// StateColor holds the string state color value
type StateColor string

// StateColorCode holds the numerical state color value
type StateColorCode int

// StateColor constant values
const (
	StateColorGray       StateColor     = "SAPControl-GRAY"
	StateColorGreen      StateColor     = "SAPControl-GREEN"
	StateColorYellow     StateColor     = "SAPControl-YELLOW"
	StateColorRed        StateColor     = "SAPControl-RED"
	StateColorCodeGray   StateColorCode = 1
	StateColorCodeGreen  StateColorCode = 2
	StateColorCodeYellow StateColorCode = 3
	StateColorCodeRed    StateColorCode = 4
)

// GetAlertTreeResponse is an xml response struct
type GetAlertTreeResponse struct {
	XMLName   xml.Name     `xml:"urn:SAPControl GetAlertTreeResponse"`
	AlertNode []*AlertNode `xml:"tree>item" json:"tree>item"`
}

// AlertNode is an xml response struct
type AlertNode struct {
	Name        string     `xml:"name" json:"name"`
	ActualValue StateColor `xml:"ActualValue" json:"ActualValue"`
	Description string     `xml:"description" json:"description"`
}

// GetInstancePropertiesResponse is an xml response struct
type GetInstancePropertiesResponse struct {
	XMLName    xml.Name            `xml:"urn:SAPControl GetInstancePropertiesResponse"`
	Properties []*InstanceProperty `xml:"properties>item" json:"properties>item"`
}

// InstanceProperty is an xml response struct
type InstanceProperty struct {
	Property string `xml:"property" json:"property"`
	Value    string `xml:"value" json:"value"`
}

// CurrentSapInstance is a struct to hold the instance xml instant property response values
type CurrentSapInstance struct {
	SID      string
	Number   int32
	Name     string
	Hostname string
}

// GetSystemInstanceListResponse is an xml response struct
type GetSystemInstanceListResponse struct {
	XMLName  xml.Name            `xml:"urn:SAPControl GetSystemInstanceListResponse"`
	Instance *ArrayOfSAPInstance `xml:"instance" json:"instance"`
}

// ArrayOfSAPInstance is an xml response struct
type ArrayOfSAPInstance struct {
	Item []*SAPInstance `xml:"item" json:"item"`
}

// SAPInstance is an xml response struct
type SAPInstance struct {
	Hostname      string     `xml:"hostname" json:"hostname"`
	InstanceNr    int32      `xml:"instanceNr" json:"instanceNr"`
	StartPriority string     `xml:"startPriority" json:"startPriority"`
	Features      string     `xml:"features" json:"features"`
	Dispstatus    StateColor `xml:"dispstatus" json:"dispstatus"`
}

// GetProcessListResponse is an xml response struct
type GetProcessListResponse struct {
	XMLName xml.Name          `xml:"urn:SAPControl GetProcessListResponse"`
	Process *ArrayOfOSProcess `xml:"process" json:"process"`
}

// ArrayOfOSProcess is an xml response struct
type ArrayOfOSProcess struct {
	Item []*OSProcess `xml:"item" json:"item"`
}

// OSProcess is an xml response struct
type OSProcess struct {
	Name        string      `xml:"name" json:"name"`
	Description string      `xml:"description" json:"description"`
	Dispstatus  *StateColor `xml:"dispstatus" json:"dispstatus"`
	Textstatus  string      `xml:"textstatus" json:"textstatus"`
	Starttime   string      `xml:"starttime" json:"starttime"`
	Elapsedtime string      `xml:"elapsedtime" json:"elapsedtime"`
	Pid         int32       `xml:"pid" json:"pid"`
}

// GetQueueStatisticResponse is an xml response struct
type GetQueueStatisticResponse struct {
	XMLName xml.Name                 `xml:"urn:SAPControl GetQueueStatisticResponse"`
	Queue   *ArrayOfTaskHandlerQueue `xml:"queue" json:"queue"`
}

// ArrayOfTaskHandlerQueue is an xml response struct
type ArrayOfTaskHandlerQueue struct {
	Item []*TaskHandlerQueue `xml:"item" json:"item"`
}

// TaskHandlerQueue is an xml response struct
type TaskHandlerQueue struct {
	Typ  string `xml:"Typ" json:"Typ"`
	Now  int32  `xml:"Now" json:"Now"`
	High int32  `xml:"High" json:"High"`
	Max  int32  `xml:"Max" json:"Max"`
}

// ABAPGetSystemWPTableResponse is an xml response struct
type ABAPGetSystemWPTableResponse struct {
	XMLName     xml.Name                  `xml:"urn:SAPControl ABAPGetSystemWPTableResponse"`
	Workprocess *ArrayOfSystemWorkProcess `xml:"workprocess" json:"workprocess"`
}

// ArrayOfSystemWorkProcess is an xml response struct
type ArrayOfSystemWorkProcess struct {
	Item []*SystemWorkProcess `xml:"item" json:"item"`
}

// SystemWorkProcess is an xml response struct
type SystemWorkProcess struct {
	Instance string `xml:"Instance" json:"Instance"`
	No       int32  `xml:"No" json:"No"`
	Typ      string `xml:"Typ" json:"Typ"`
	Pid      int32  `xml:"Pid" json:"Pid"`
	Status   string `xml:"Status" json:"Status"`
	Reason   string `xml:"Reason" json:"Reason"`
	Start    string `xml:"Start" json:"Start"`
	Err      string `xml:"Err" json:"Err"`
	Sem      string `xml:"Sem" json:"Sem"`
	CPU      string `xml:"Cpu" json:"Cpu"`
	Time     string `xml:"Time" json:"Time"`
	Program  string `xml:"Program" json:"Program"`
	Client   string `xml:"Client" json:"Client"`
	User     string `xml:"User" json:"User"`
	Action   string `xml:"Action" json:"Action"`
	Table    string `xml:"Table" json:"Table"`
}

// GetRequestLogFileResponse is an xml response struct
type GetRequestLogFileResponse struct {
	XMLName xml.Name  `xml:"urn:SAPControl GetRequestLogFileResponse"`
	Content []*string `xml:"content" json:"content"`
}

// OSExecuteResponse is an xml response struct
type OSExecuteResponse struct {
	XMLName  xml.Name       `xml:"urn:SAPControl OSExecuteResponse"`
	Exitcode int32          `xml:"exitcode" json:"exitcode"`
	Pid      int32          `xml:"pid" json:"pid"`
	Lines    *ArrayOfString `xml:"lines" json:"lines"`
}

// ArrayOfString is an xml response struct
type ArrayOfString struct {
	Item []string `xml:"item" json:"item"`
}

// EnqGetStatisticResponse is an xml response struct
type EnqGetStatisticResponse struct {
	XMLName       xml.Name `xml:"urn:SAPControl EnqStatistic"`
	LocksNow      *int32   `xml:"locks-now" json:"locks-now"`
	LocksHigh     *int32   `xml:"locks-high" json:"locks-high"`
	LocksMax      *int32   `xml:"locks-max" json:"locks-max"`
	EnqueueErrors *int64   `xml:"enqueue-errors" json:"enqueue-errors"`
	DequeueErrors *int64   `xml:"dequeue-errors" json:"dequeue-errors"`
	LockTime      *float64 `xml:"lock-time" json:"lock-time"`
	LockWaitTime  *float64 `xml:"lock-wait-time" json:"lock-wait-time"`
}
