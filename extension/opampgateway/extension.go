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

package opampgateway

import (
	"context"

	"github.com/observiq/bindplane-otel-collector/extension/opampgateway/internal/gateway"
	"go.opentelemetry.io/collector/component"
)

// OpAMPGateway is the OTel extension wrapper around the internal gateway implementation.
type OpAMPGateway struct {
	gateway           *gateway.Gateway
	telemetrySettings component.TelemetrySettings
}

// Start starts the OpAMP gateway extension.
func (o *OpAMPGateway) Start(ctx context.Context, host component.Host) error {
	return o.gateway.Start(ctx, host, o.telemetrySettings)
}

// Shutdown stops the OpAMP gateway extension.
func (o *OpAMPGateway) Shutdown(ctx context.Context) error {
	return o.gateway.Shutdown(ctx)
}
