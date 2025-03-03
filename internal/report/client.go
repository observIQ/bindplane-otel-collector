// Copyright  Bindplane, Inc.
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

package report

import (
	"net/http"
)

// Client represents a client that can report information to a platform
//
//go:generate mockery --name Client --filename mock_client.go --structname MockClient
type Client interface {
	// Do makes the specified request and return the body contents.
	Do(req *http.Request) (*http.Response, error)
}
