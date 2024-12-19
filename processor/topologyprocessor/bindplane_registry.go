// // Copyright observIQ, Inc.
// //
// // Licensed under the Apache License, Version 2.0 (the "License");
// // you may not use this file except in compliance with the License.
// // You may obtain a copy of the License at
// //
// //     http://www.apache.org/licenses/LICENSE-2.0
// //
// // Unless required by applicable law or agreed to in writing, software
// // distributed under the License is distributed on an "AS IS" BASIS,
// // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// // See the License for the specific language governing permissions and
// // limitations under the License.

//go:build bindplane

package topologyprocessor

import (
	"github.com/observiq/bindplane-otel-collector/internal/topology"
	"go.opentelemetry.io/collector/component"
)

// GetTopologyRegistry returns the topology registry that should be registered to based on the component ID.
// nil, nil may be returned by this function. In this case, the processor should not register it's topology state anywhere.
func GetTopologyRegistry(host component.Host, bindplane component.ID) (topology.TopoRegistry, error) {
	return topology.BindplaneAgentTopologyRegistry, nil
}
