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

package chronicleexporter

import (
	"sort"
	"strings"

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
)

type logGroup struct {
	namespace       string
	logType         string
	ingestionLabels []*api.Label
	entries         []*api.LogEntry
}

type logGrouper struct {
	// Use composite key to group logs
	groups map[string]*logGroup
}

func newLogGrouper() *logGrouper {
	return &logGrouper{
		groups: make(map[string]*logGroup),
	}
}

// createGroupKey creates a unique key for grouping logs based on namespace, logType and ingestionLabels
func createGroupKey(namespace, logType string, ingestionLabels []*api.Label) string {
	// Start with namespace and logType
	key := namespace + "|" + logType

	// Sort and append labels to maintain consistent key ordering
	if len(ingestionLabels) > 0 {
		labelStrs := make([]string, len(ingestionLabels))
		for i, label := range ingestionLabels {
			labelStrs[i] = label.Key + "=" + label.Value
		}
		sort.Strings(labelStrs)
		key += "|" + strings.Join(labelStrs, ",")
	}

	return key
}

// Add adds a log entry to the appropriate group
func (g *logGrouper) Add(log *api.LogEntry, namespace, logType string, ingestionLabels []*api.Label) {
	key := createGroupKey(namespace, logType, ingestionLabels)

	if group, exists := g.groups[key]; exists {
		group.entries = append(group.entries, log)
	} else {
		g.groups[key] = &logGroup{
			namespace:       namespace,
			logType:         logType,
			ingestionLabels: ingestionLabels,
			entries:         []*api.LogEntry{log},
		}
	}
}

// ForEach iterates over all groups and calls the provided function for each group
func (g *logGrouper) ForEach(fn func(entries []*api.LogEntry, namespace, logType string, ingestionLabels []*api.Label)) {
	for _, group := range g.groups {
		fn(group.entries, group.namespace, group.logType, group.ingestionLabels)
	}
}

func (g *logGrouper) Clear() {
	g.groups = make(map[string]*logGroup)
}
