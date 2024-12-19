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

package topologyprocessor

import (
	"fmt"

	"github.com/observiq/bindplane-otel-collector/internal/topology"
	"go.opentelemetry.io/collector/component"
)

// GetTopologyRegistry returns the topology registry that should be registered to based on the component ID.
// nil, nil may be returned by this function. In this case, the processor should not register it's topology state anywhere.
func GetTopologyRegistry(host component.Host, bindplane component.ID) (topology.TopoRegistry, error) {
	var emptyComponentID component.ID
	if bindplane == emptyComponentID {
		// No bindplane component referenced, so we won't register our topology state anywhere.
		return nil, nil
	}

	ext, ok := host.GetExtensions()[bindplane]
	if !ok {
		return nil, fmt.Errorf("bindplane extension %q does not exist", bindplane)
	}

	registry, ok := ext.(topology.TopoRegistry)
	if !ok {
		return nil, fmt.Errorf("extension %q is not an topology state registry", bindplane)
	}

	return registry, nil
}
