// Copyright The OpenTelemetry Authors
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

package logsreceiver

//revive:disable:blank-imports These imports are all necessary to register operators for use; They should not be in a main package.
import (
	// Register parsers and transformers for stanza-based log receivers
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/input/file"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/input/generate"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/input/k8sevent"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/input/stdin"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/input/syslog"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/input/tcp"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/input/udp"

	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/csv"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/json"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/regex"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/severity"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/time"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/trace"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/parser/uri"

	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/add"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/copy"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/filter"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/flatten"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/metadata"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/noop"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/recombine"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/remove"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/restructure"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/retain"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/transformer/router"

	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/output/drop"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/output/file"
	_ "github.com/open-telemetry/opentelemetry-log-collection/operator/builtin/output/stdout"

	// register non-opentelemetry operators
	_ "github.com/observiq/observiq-collector/receiver/operators/input/aws/cloudwatch"
	_ "github.com/observiq/observiq-collector/receiver/operators/input/azure/eventhub"
	_ "github.com/observiq/observiq-collector/receiver/operators/input/azure/loganalytics"
	_ "github.com/observiq/observiq-collector/receiver/operators/input/goflow"
	_ "github.com/observiq/observiq-collector/receiver/operators/input/http"

	_ "github.com/observiq/observiq-collector/receiver/operators/parser/keyvalue"

	_ "github.com/observiq/observiq-collector/receiver/operators/transformer/k8smetadata"
)
