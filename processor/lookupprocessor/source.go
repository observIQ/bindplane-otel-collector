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

package lookupprocessor

// LookupSource is an interface for different lookup data sources
type LookupSource interface {
	// Lookup returns a map of attributes for the given key
	Lookup(key string) (map[string]string, error)
	// Load initializes or refreshes the lookup source
	Load() error
	// Close cleans up resources used by the lookup source
	Close() error
}
