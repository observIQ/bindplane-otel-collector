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

// Reporter represents a a structure to collector and report specific structures
//
//go:generate mockery --name Reporter --filename mock_reporter.go --structname MockReporter
type Reporter interface {
	// Kind returns the kind of this reporter
	Kind() string

	// Report starts reporting with the passed in configuration.
	Report(config any) error
}
