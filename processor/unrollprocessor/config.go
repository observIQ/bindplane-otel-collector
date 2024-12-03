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

// Package unrollprocessor contains the logic to unroll logs from a slice in the body field.
package unrollprocessor

import (
	"errors"
)

// Config is the configuration for the unroll processor.
type Config struct {
	Field     UnrollField `mapstructure:"field"`
	Recursive bool        `mapstructure:"recursive"`
}

// UnrollField is the field to unroll.
type UnrollField string

const (
	// UnrollFieldBody is the only supported field for unrolling logs.
	UnrollFieldBody UnrollField = "body"
)

// Validate checks the configuration for any issues.
func (c *Config) Validate() error {
	if c.Field != UnrollFieldBody {
		return errors.New("only unrolling logs from a body slice is currently supported")
	}

	return nil
}
