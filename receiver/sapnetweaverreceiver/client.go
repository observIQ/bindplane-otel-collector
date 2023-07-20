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

// Package sapnetweaverreceiver is an OTel compatible receiver
package sapnetweaverreceiver // import "github.com/observiq/bindplane-agent/receiver/sapnetweaverreceiver"

import (
	"fmt"

	"go.opentelemetry.io/collector/component"

	"github.com/hooklift/gowsdl/soap"
)

func newSoapClient(cfg *Config, host component.Host, settings component.TelemetrySettings) (*soap.Client, error) {
	httpClient, err := cfg.ToClient(host, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP Client: %w", err)
	}

	return soap.NewClient(cfg.Endpoint, soap.WithBasicAuth(cfg.Username, cfg.Password), soap.WithHTTPClient(httpClient)), nil
}
