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

// Package resourceattributetransposerprocessor provides a processor that transposes resource attributes to datapoint attributes
package resourceattributetransposerprocessor

import "go.opentelemetry.io/collector/config"

// CopyResourceConfig is a config struct specifying a mapping of a resource attribute to a datapoint attribute
type CopyResourceConfig struct {
	// From is the attribute on the resource to copy from
	From string `mapstructure:"from"`
	// To is the attribute to copy to on the individual data point
	To string `mapstructure:"to"`
}

// Config is the configuration for the resourceattributetransposer
type Config struct {
	config.ProcessorSettings `mapstructure:",squash"`
	// Operations is a list of copy operations to perform on each ResourceMetric.
	Operations []CopyResourceConfig `mapstructure:"operations"`
}
