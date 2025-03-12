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

package bindplaneauditlogs

import "time"

// AuditLogEvent represents an event effecting a resource that is used to create an audit trail
type AuditLogEvent struct {
	// ID is a ULID uniquely identifying this event
	ID string `json:"id"`
	// Timestamp is the time this event occurred
	Timestamp *time.Time `json:"timestamp"`
	// ResourceName is the resource name + friendly name of the resource
	ResourceName string `json:"resourceName"`
	// Description is the friendly name of the resource
	Description string `json:"description" csv:"description"`
	// ResourceKind is the resource that was modified
	ResourceKind string `json:"resourceKind"`
	// Configuration is the name of the configuration affected. This may be nil if there is not associated configuration.
	Configuration string `json:"configuration,omitempty"`
	// Action is the action that was taken on the resource
	Action string `json:"action"`
	// User is the user that modified the resource
	User string `json:"user"`
	// Account is the account that the event occurred on. May be nil in the case of single-account.
	Account string `json:"account,omitempty"`
}
