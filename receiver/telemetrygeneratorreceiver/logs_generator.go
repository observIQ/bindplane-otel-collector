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

package telemetrygeneratorreceiver //import "github.com/observiq/bindplane-agent/receiver/telemetrygeneratorreceiver"

import (
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// defaultLogGenerator is a generator for logs. It generates a stream of logs based on the configuration provided,
// each log identical save for the timestamp.
type defaultLogGenerator struct {
	cfg    GeneratorConfig
	logs   plog.Logs
	logger *zap.Logger
}

func newLogsGenerator(cfg GeneratorConfig, logger *zap.Logger) logGenerator {
	lg := &defaultLogGenerator{
		cfg:    cfg,
		logger: logger,
		logs:   plog.NewLogs(),
	}

	// Add resource attributes
	resourceLogs := lg.logs.ResourceLogs().AppendEmpty()
	err := resourceLogs.Resource().Attributes().FromRaw(lg.cfg.ResourceAttributes)
	if err != nil {
		// validation should catch this error
		logger.Warn("Error adding resource attributes", zap.Error(err))
	}

	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecord := scopeLogs.LogRecords().AppendEmpty()

	err = logRecord.Attributes().FromRaw(lg.cfg.Attributes)
	if err != nil {
		// validation should catch this error
		logger.Warn("Error adding attributes", zap.Error(err))
	}
	for k, v := range lg.cfg.AdditionalConfig {
		switch k {
		case "body":
			// validation already proves this is a string
			logRecord.Body().SetStr(v.(string))
		case "severity":
			// validation already proves this is an int
			logRecord.SetSeverityNumber(plog.SeverityNumber(v.(int)))
		}
	}
	return lg
}

func (g *defaultLogGenerator) generateLogs() plog.Logs {
	for i := 0; i < g.logs.ResourceLogs().Len(); i++ {
		resourceLogs := g.logs.ResourceLogs().At(i)
		for k := 0; k < resourceLogs.ScopeLogs().Len(); k++ {
			scopeLogs := resourceLogs.ScopeLogs().At(k)
			for j := 0; j < scopeLogs.LogRecords().Len(); j++ {
				log := scopeLogs.LogRecords().At(j)
				log.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
			}
		}
	}
	return g.logs
}
