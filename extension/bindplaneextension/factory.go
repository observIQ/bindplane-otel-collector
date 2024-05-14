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

package bindplaneextension

import (
	"context"

	"github.com/observiq/bindplane-agent/extension/bindplaneextension/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

// NewFactory creates a new factory for the bindplane extension
func NewFactory() extension.Factory {
	return extension.NewFactory(
		metadata.Type,
		defaultConfig,
		createBindPlaneExtension,
		metadata.ExtensionStability,
	)
}

func defaultConfig() component.Config {
	return &Config{}
}

func createBindPlaneExtension(_ context.Context, _ extension.CreateSettings, _ component.Config) (extension.Extension, error) {
	return bindplaneExtension{}, nil
}
