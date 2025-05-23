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

//go:build !bindplane

package throughputmeasurementprocessor

import (
	"fmt"

	"github.com/observiq/bindplane-otel-collector/internal/measurements"
	"go.opentelemetry.io/collector/component"
)

// GetThroughputRegistry returns the throughput registry that should be registered to based on the component ID.
// nil, nil may be returned by this function. In this case, the processor should not register it's throughput measurements anywhere.
func GetThroughputRegistry(host component.Host, bindplane component.ID) (measurements.ThroughputMeasurementsRegistry, error) {
	var emptyComponentID component.ID
	if bindplane == emptyComponentID {
		// No bindplane component referenced, so we won't register our measurements anywhere.
		return nil, nil
	}

	ext, ok := host.GetExtensions()[bindplane]
	if !ok {
		return nil, fmt.Errorf("bindplane extension %q does not exist", bindplane)
	}

	registry, ok := ext.(measurements.ThroughputMeasurementsRegistry)
	if !ok {
		return nil, fmt.Errorf("extension %q is not an throughput message registry", bindplane)
	}

	return registry, nil
}
