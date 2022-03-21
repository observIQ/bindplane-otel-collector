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

package varnishreceiver // import "github.com/observiq/observiq-otel-collector/receiver/varnishreceiver"

import (
	"errors"
	"fmt"
	"os"

	"go.opentelemetry.io/collector/receiver/scraperhelper"
	"go.uber.org/multierr"

	"github.com/observiq/observiq-otel-collector/receiver/varnishreceiver/internal/metadata"
)

var (
	errWorkingDirNotExist = errors.New(`"working_dir" does not exists %q`)
	errExecDirNotExist    = errors.New(`"exec_dir" does not exists %q`)
)

// Config defines configuration for varnish metrics receiver.
type Config struct {
	scraperhelper.ScraperControllerSettings `mapstructure:",squash"`
	Metrics                                 metadata.MetricsSettings `mapstructure:"metrics"`
	WorkingDir                              string                   `mapstructure:"working_dir"`
	ExecDir                                 string                   `mapstructure:"exec_dir"`
}

// Validate validates the config.
func (c *Config) Validate() error {
	var err error
	if c.WorkingDir != "" {
		if _, pathErr := os.Stat(c.WorkingDir); pathErr != nil {
			err = multierr.Append(err, fmt.Errorf(errWorkingDirNotExist.Error(), pathErr))
		}
	}
	if c.ExecDir != "" {
		if _, pathErr := os.Stat(c.ExecDir); pathErr != nil {
			err = multierr.Append(err, fmt.Errorf(errExecDirNotExist.Error(), pathErr))
		}
	}

	return err
}
