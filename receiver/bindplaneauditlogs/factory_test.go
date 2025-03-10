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

package bindplaneauditlogs

import (
	"context"
	"testing"

	"github.com/observiq/bindplane-otel-collector/receiver/bindplaneauditlogs/internal/metadata"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestType(t *testing.T) {
	factory := NewFactory()
	ft := factory.Type()
	require.EqualValues(t, metadata.Type, ft)
}

func TestCreateLogsReceiver(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	cfg.APIKey = "test-api-key"
	cfg.Scheme = "https"
	cfg.ClientConfig.Endpoint = "https://test.bindplane.com"

	// Need to validate to set up the bindplaneURL
	err := cfg.Validate()
	require.NoError(t, err)

	recv, err := NewFactory().CreateLogs(
		context.Background(),
		receivertest.NewNopSettings(),
		cfg,
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, recv)
}
