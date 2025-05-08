// Copyright observIQ, Inc.
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

package grpcq

import (
	"fmt"
	"hash/fnv"
	"sort"

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
)

// computeGroupKey creates a unique key for grouping logs based on namespace, logType and ingestionLabels
func computeGroupKey(batch *api.LogEntryBatch) string {
	h := fnv.New64a()
	h.Write([]byte(batch.GetLogType()))
	h.Write([]byte(batch.GetSource().GetNamespace()))

	labels := batch.GetSource().GetLabels()
	if len(labels) == 0 {
		return fmt.Sprintf("%x", h.Sum64())
	}

	pairs := make([]string, 0, len(labels))
	for _, label := range labels {
		pairs = append(pairs, fmt.Sprintf("%s=%s", label.Key, label.Value))
	}

	sort.Strings(pairs)

	for _, pair := range pairs {
		h.Write([]byte(pair))
	}
	return fmt.Sprintf("%x", h.Sum64())
}
