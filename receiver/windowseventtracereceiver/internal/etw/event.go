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

package etw

import "time"

// Event is a struct that represents an event from the ETW session which is pre-parsed into a more usable format
type Event struct {
	Flags struct {
		// Use to flag event as being skippable for performance reason
		Skippable bool
	} `json:"-"`

	EventData map[string]interface{} `json:",omitempty"`
	UserData  map[string]interface{} `json:",omitempty"`
	System    struct {
		Channel     string
		Computer    string
		EventID     string `json:",omitempty"`
		EventType   string `json:",omitempty"`
		EventGUID   string `json:",omitempty"`
		Correlation struct {
			ActivityID        string
			RelatedActivityID string
		}
		Execution struct {
			ProcessID uint32
			ThreadID  uint32
		}
		Keywords struct {
			Value uint64
			Name  string
		}
		Level struct {
			Value uint8
			Name  string
		}
		Opcode struct {
			Value uint8
			Name  string
		}
		Task struct {
			Value uint8
			Name  string
		}
		Provider struct {
			GUID string
			Name string
		}
		TimeCreated struct {
			SystemTime time.Time
		}
	}
	ExtendedData []string `json:",omitempty"`
}
