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

package azure

import (
	"fmt"

	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/open-telemetry/opentelemetry-log-collection/operator/helper"
)

// Config is the configuration of a Azure Event Hub input operator.
type Config struct {
	helper.InputOperator

	// required
	Namespace        string `json:"namespace,omitempty"         yaml:"namespace,omitempty"`
	Name             string `json:"name,omitempty"              yaml:"name,omitempty"`
	Group            string `json:"group,omitempty"             yaml:"group,omitempty"`
	ConnectionString string `json:"connection_string,omitempty" yaml:"connection_string,omitempty"`

	// optional
	PrefetchCount uint32 `json:"prefetch_count,omitempty" yaml:"prefetch_count,omitempty"`
	StartAt       string `json:"start_at,omitempty"       yaml:"start_at,omitempty"`

	startAtBeginning bool
}

// Build builds the event hub input operator
func (a *Config) Build(buildContext operator.BuildContext, input helper.InputConfig) error {
	inputOperator, err := input.Build(buildContext)
	if err != nil {
		return err
	}
	a.InputOperator = inputOperator

	switch a.StartAt {
	case "beginning":
		a.startAtBeginning = true
	case "end":
		a.startAtBeginning = false
	}

	return a.validate()
}

func (a Config) validate() error {
	if a.Namespace == "" {
		return fmt.Errorf("missing required parameter 'namespace'")
	}

	if a.Name == "" {
		return fmt.Errorf("missing required parameter 'name'")
	}

	if a.Group == "" {
		return fmt.Errorf("missing required parameter 'group'")
	}

	if a.ConnectionString == "" {
		return fmt.Errorf("missing required parameter 'connection_string'")
	}

	if a.PrefetchCount < 1 {
		return fmt.Errorf("invalid value for parameter 'prefetch_count'")
	}

	if a.StartAt != "beginning" && a.StartAt != "end" {
		return fmt.Errorf("invalid value for parameter 'start_at'")
	}

	return nil
}
