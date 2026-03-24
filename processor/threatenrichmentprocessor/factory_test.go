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

package threatenrichmentprocessor

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/processor/processortest"
)

func TestNewFactory(t *testing.T) {
	f := NewFactory()
	require.Equal(t, componentType, f.Type())
	require.Equal(t, stability, f.LogsStability())
	require.NotNil(t, f.CreateDefaultConfig())
	require.NotNil(t, f.CreateLogs)
}

func TestCreateDefaultConfig(t *testing.T) {
	cfg := NewFactory().CreateDefaultConfig().(*Config)
	require.IsType(t, &Config{}, cfg)
}

func TestCreateLogsProcessor_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	indFile := filepath.Join(dir, "i.txt")
	require.NoError(t, writeIndicatorFileText(indFile, "bad.com\n"))

	cfg := validBloomConfig()
	cfg.Rules[0].IndicatorFile = indFile
	require.NoError(t, cfg.Validate())

	f := NewFactory()
	set := processortest.NewNopSettings(componentType)
	p, err := f.CreateLogs(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	require.NotNil(t, p)
	require.NoError(t, p.Start(context.Background(), nil))
	require.NoError(t, p.Shutdown(context.Background()))
}

func TestCreateLogsProcessor_InvalidConfigType(t *testing.T) {
	f := NewFactory()
	set := processortest.NewNopSettings(componentType)
	_, err := f.CreateLogs(context.Background(), set, nil, consumertest.NewNop())
	require.ErrorIs(t, err, errInvalidConfigType)
}

func writeIndicatorFileText(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
