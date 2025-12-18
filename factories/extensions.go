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

package factories

import (
	"github.com/observiq/bindplane-otel-collector/extension/awss3eventextension"
	"github.com/observiq/bindplane-otel-collector/extension/badgerextension"
	"github.com/observiq/bindplane-otel-collector/extension/pebbleextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/basicauthextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/bearertokenauthextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/cgroupruntimeextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding/avrologencodingextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding/googlecloudlogentryencodingextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding/jsonlogencodingextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding/textencodingextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/headerssetterextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/healthcheckextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/oauth2clientauthextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/oidcauthextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/pprofextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/sigv4authextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/storage/filestorage"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/storage/redisstorageextension"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/extension/zpagesextension"
)

var defaultExtensions = []extension.Factory{
	avrologencodingextension.NewFactory(),
	awss3eventextension.NewFactory(),
	badgerextension.NewFactory(),
	basicauthextension.NewFactory(),
	bearertokenauthextension.NewFactory(),
	cgroupruntimeextension.NewFactory(),
	extensiontest.NewNopFactory(),
	filestorage.NewFactory(),
	headerssetterextension.NewFactory(),
	healthcheckextension.NewFactory(),
	googlecloudlogentryencodingextension.NewFactory(),
	jsonlogencodingextension.NewFactory(),
	textencodingextension.NewFactory(),
	pebbleextension.NewFactory(),
	oauth2clientauthextension.NewFactory(),
	oidcauthextension.NewFactory(),
	pprofextension.NewFactory(),
	redisstorageextension.NewFactory(),
	sigv4authextension.NewFactory(),
	zpagesextension.NewFactory(),
}
