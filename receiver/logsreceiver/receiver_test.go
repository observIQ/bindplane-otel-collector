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

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-log-collection/entry"
	"github.com/open-telemetry/opentelemetry-log-collection/pipeline"
	"github.com/open-telemetry/opentelemetry-log-collection/testutil"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"gopkg.in/yaml.v2"
)

func TestStart(t *testing.T) {
	params := component.ReceiverCreateSettings{
		Logger: zaptest.NewLogger(t),
	}
	mockConsumer := mockLogsConsumer{}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Pipeline = []map[string]interface{}{
		{
			"type": "json_parser",
		},
	}
	ctx := context.Background()
	logsReceiver, err := factory.CreateLogsReceiver(
		ctx,
		params,
		cfg,
		&mockConsumer,
	)
	require.NoError(t, err, "receiver should successfully build")

	err = logsReceiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err, "receiver start failed")

	stanzaReceiver := logsReceiver.(*receiver)
	stanzaReceiver.emitter.logChan <- entry.New()

	// Eventually because of asynchronuous nature of the receiver.
	require.Eventually(t,
		func() bool {
			return mockConsumer.Received() == 1
		},
		10*time.Second, 5*time.Millisecond, "one log entry expected",
	)
	require.NoError(t, logsReceiver.Shutdown(ctx))
}

func TestPlugins(t *testing.T) {
	params := component.ReceiverCreateSettings{
		Logger: zaptest.NewLogger(t),
	}
	mockConsumer := mockLogsConsumer{}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Pipeline = []map[string]interface{}{
		{
			"type": "hello",
		},
	}
	cfg.PluginDir = filepath.Join("testdata", "plugins")
	ctx := context.Background()
	logsReceiver, err := factory.CreateLogsReceiver(
		ctx,
		params,
		cfg,
		&mockConsumer,
	)
	require.NoError(t, err, "receiver should successfully build")

	// Plugin should emit one entry
	err = logsReceiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err, "receiver start failed")

	// Eventually because of asynchronuous nature of the receiver.
	require.Eventually(t,
		func() bool {
			return mockConsumer.Received() == 1
		},
		10*time.Second, 5*time.Millisecond, "one log entry expected",
	)
	require.NoError(t, logsReceiver.Shutdown(ctx))
}

func TestHandleStartError(t *testing.T) {
	params := component.ReceiverCreateSettings{
		Logger: zaptest.NewLogger(t),
	}
	mockConsumer := mockLogsConsumer{}

	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Pipeline = []map[string]interface{}{
		{
			"type": "unstartable_operator", // TODO pipeline should require an input operator
		},
	}

	receiver, err := factory.CreateLogsReceiver(context.Background(), params, cfg, &mockConsumer)
	require.NoError(t, err, "receiver should successfully build")

	err = receiver.Start(context.Background(), componenttest.NewNopHost())
	require.Error(t, err, "receiver fails to start under rare circumstances")
}

func TestHandleConsumeError(t *testing.T) {
	params := component.ReceiverCreateSettings{
		Logger: zaptest.NewLogger(t),
	}
	mockConsumer := mockLogsRejecter{}
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Pipeline = []map[string]interface{}{
		{
			"type": "json_parser", // TODO pipeline should require an input operator
		},
	}

	ctx := context.Background()
	logsReceiver, err := factory.CreateLogsReceiver(ctx, params, cfg, &mockConsumer)
	require.NoError(t, err, "receiver should successfully build")

	err = logsReceiver.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err, "receiver start failed")

	stanzaReceiver := logsReceiver.(*receiver)
	stanzaReceiver.emitter.logChan <- entry.New()

	// Eventually because of asynchronuous nature of the receiver.
	require.Eventually(t,
		func() bool {
			return mockConsumer.Rejected() == 1
		},
		10*time.Second, 5*time.Millisecond, "one log entry expected",
	)
	require.NoError(t, logsReceiver.Shutdown(ctx))
}

func BenchmarkReadLine(b *testing.B) {

	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		b.Errorf(err.Error())
		b.FailNow()
	}

	filePath := filepath.Join(tempDir, "bench.log")

	pipelineYaml := fmt.Sprintf(`
- type: file_input
  include:
    - %s
  start_at: beginning`,
		filePath)

	pipelineCfg := pipeline.Config{}
	require.NoError(b, yaml.Unmarshal([]byte(pipelineYaml), &pipelineCfg))

	emitter := NewLogEmitter(zap.NewNop().Sugar())
	defer func() {
		require.NoError(b, emitter.Stop())
	}()

	buildContext := testutil.NewBuildContext(b)
	pl, err := pipelineCfg.BuildPipeline(buildContext, emitter)
	require.NoError(b, err)

	// Populate the file that will be consumed
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0666)
	require.NoError(b, err)
	for i := 0; i < b.N; i++ {
		_, err = file.WriteString("testlog\n")
		require.NoError(b, err)
	}

	// // Run the actual benchmark
	b.ResetTimer()
	require.NoError(b, pl.Start(newMockPersister()))
	for i := 0; i < b.N; i++ {
		convert(<-emitter.logChan)
	}
}

func BenchmarkParseAndMap(b *testing.B) {

	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		b.Errorf(err.Error())
		b.FailNow()
	}

	filePath := filepath.Join(tempDir, "bench.log")

	fileInputYaml := fmt.Sprintf(`
- type: file_input
  include:
    - %s
  start_at: beginning`, filePath)

	regexParserYaml := `
- type: regex_parser
  regex: '(?P<remote_host>[^\s]+) - (?P<remote_user>[^\s]+) \[(?P<timestamp>[^\]]+)\] "(?P<http_method>[A-Z]+) (?P<path>[^\s]+)[^"]+" (?P<http_status>\d+) (?P<bytes_sent>[^\s]+)'
  timestamp:
    parse_from: timestamp
    layout: '%d/%b/%Y:%H:%M:%S %z'
  severity:
    parse_from: http_status
    preserve: true
    mapping:
      critical: 5xx
      error: 4xx
      info: 3xx
      debug: 2xx`

	pipelineYaml := fmt.Sprintf("%s%s", fileInputYaml, regexParserYaml)

	pipelineCfg := pipeline.Config{}
	require.NoError(b, yaml.Unmarshal([]byte(pipelineYaml), &pipelineCfg))

	emitter := NewLogEmitter(zap.NewNop().Sugar())
	defer func() {
		require.NoError(b, emitter.Stop())
	}()

	buildContext := testutil.NewBuildContext(b)
	pl, err := pipelineCfg.BuildPipeline(buildContext, emitter)
	require.NoError(b, err)

	// Populate the file that will be consumed
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0666)
	require.NoError(b, err)
	for i := 0; i < b.N; i++ {
		str := fmt.Sprintf("10.33.121.119 - - [11/Aug/2020:00:00:00 -0400] \"GET /index.html HTTP/1.1\" 404 %d\n", i%1000)
		_, err = file.WriteString(str)
		require.NoError(b, err)
	}

	// // Run the actual benchmark
	b.ResetTimer()
	require.NoError(b, pl.Start(newMockPersister()))
	for i := 0; i < b.N; i++ {
		convert(<-emitter.logChan)
	}
}

func testdataRotateTestYamlAsMap(tempDir string) *Config {
	return &Config{
		ReceiverSettings: config.NewReceiverSettings(config.NewID(typeStr)),
		Pipeline: OperatorConfigs{
			map[string]interface{}{
				"type": "file_input",
				"include": []interface{}{
					fmt.Sprintf("%s/*", tempDir),
				},
				"include_file_name": false,
				"poll_interval":     "10ms",
				"start_at":          "beginning",
			},
			map[string]interface{}{
				"type":  "regex_parser",
				"regex": "^(?P<ts>\\d{4}-\\d{2}-\\d{2}) (?P<msg>[^\n]+)",
				"timestamp": map[interface{}]interface{}{
					"layout":     "%Y-%m-%d",
					"parse_from": "ts",
				},
			},
		},
		Converter: ConverterConfig{
			MaxFlushCount: DefaultMaxFlushCount,
			FlushInterval: DefaultFlushInterval,
		},
	}
}
