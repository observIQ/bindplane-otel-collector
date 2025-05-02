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

//go:build windows

// Package etw contains the functionality for interacting with the ETW API.
package etw

import (
	"time"

	"golang.org/x/sys/windows"
)

// EventFlags contains flags for the event
type EventFlags struct {
	// Use to flag event as being skippable for performance reason
	Skippable bool `json:"skippable"`
}

// zeroGUID is a string representation of the zero GUID, which generally indicates that the value is not applicable
var zeroGUID = windows.GUID{}.String()

// EventCorrelation contains correlation information for the event
type EventCorrelation struct {
	ActivityID        string `json:"activityID"`
	RelatedActivityID string `json:"relatedActivityID"`
}

// EventExecution contains execution information for the event
type EventExecution struct {
	ProcessID uint32 `json:"processID"`
	ThreadID  uint32 `json:"threadID"`
}

// EventKeywords contains keyword information for the event
type EventKeywords struct {
	Value uint64 `json:"value"`
	Name  string `json:"name"`
}

// EventOpcode contains opcode information for the event
type EventOpcode struct {
	Value uint8  `json:"value"`
	Name  string `json:"name"`
}

// EventTask contains task information for the event
type EventTask struct {
	Value uint8  `json:"value"`
	Name  string `json:"name"`
}

// EventProvider contains provider information for the event
type EventProvider struct {
	GUID string `json:"guid"`
	Name string `json:"name"`
}

// EventTimeCreated contains time information for the event
type EventTimeCreated struct {
	SystemTime time.Time `json:"systemTime"`
}

// EventSystem contains system information for the event
type EventSystem struct {
	ActivityID  string           `json:"activityID"`
	Channel     string           `json:"channel"`
	Computer    string           `json:"computer"`
	EventID     string           `json:"eventID,omitempty"`
	Correlation EventCorrelation `json:"correlation"`
	Execution   EventExecution   `json:"execution"`
	Keywords    string           `json:"keywords"`
	Level       uint8            `json:"level"`
	Opcode      string           `json:"opcode"`
	Task        string           `json:"task"`
	Provider    EventProvider    `json:"provider"`
	TimeCreated EventTimeCreated `json:"timeCreated"`
	Version     uint8            `json:"version"`
}

// Event is a struct that represents an event from the ETW session which is pre-parsed into a more usable format by the receiver
type Event struct {
	Raw          string         `json:"raw,omitempty"`
	Session      string         `json:"session"`
	Flags        string         `json:"flags"`
	Timestamp    time.Time      `json:"timestamp"`
	EventData    map[string]any `json:"eventData,omitempty"`
	UserData     map[string]any `json:"userData,omitempty"`
	System       EventSystem    `json:"system"`
	ExtendedData []string       `json:"extendedData,omitempty"`
	Security     EventSecurity  `json:"security,omitempty"`
}

// EventSecurity contains security information for the event
type EventSecurity struct {
	SID string `json:"sid"`
}
