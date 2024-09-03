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

package bindplaneextension

import (
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
)

// Config is the configuration for the bindplane extension
type Config struct {
	// Labels in "k1=v1,k2=v2" format
	Labels string `mapstructure:"labels"`
	// Component ID of the opamp extension. If not specified, then
	// this extension will not generate any custom messages for throughput metrics.
	OpAMP component.ID `mapstructure:"opamp"`
	// MeasurementsInterval is the interval on which to report measurements.
	// Measurements reporting is disabled if this duration is 0.
	MeasurementsInterval time.Duration `mapstructure:"measurements_interval"`
	// ExtraMeasurementsAttributes are a map of key-value pairs to add to all reported measurements.
	ExtraMeasurementsAttributes map[string]string `mapstructure:"extra_measurements_attributes,omitempty"`
}

// Validate returns an error if the config is invalid
func (c Config) Validate() error {
	if c.MeasurementsInterval < 0 {
		return errors.New("measurements interval must be postitive or 0")
	}

	return nil
}
