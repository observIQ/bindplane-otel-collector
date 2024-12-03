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

package unrollprocessor

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

type unrollProcessor struct {
	cfg *Config
}

// newUnrollProcessor returns a new unrollProcessor.
func newUnrollProcessor(config *Config) (*unrollProcessor, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &unrollProcessor{
		cfg: config,
	}, nil
}

// ProcessLogs implements the processor interface
func (p *unrollProcessor) ProcessLogs(_ context.Context, ld plog.Logs) (plog.Logs, error) {
	var errs error
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		rls := ld.ResourceLogs().At(i)
		for j := 0; j < rls.ScopeLogs().Len(); j++ {
			sls := rls.ScopeLogs().At(j)
			origLen := sls.LogRecords().Len()
			var last func() int
			if p.cfg.Recursive {
				last = sls.LogRecords().Len
			} else {
				last = func() int { return origLen }
			}
			for k := 0; k < last(); k++ {
				lr := sls.LogRecords().At(k)
				if lr.Body().Type() != pcommon.ValueTypeSlice {
					continue
				}
				for l := 0; l < lr.Body().Slice().Len(); l++ {
					newRecord := sls.LogRecords().AppendEmpty()
					lr.CopyTo(newRecord)
					p.setBody(newRecord, lr.Body().Slice().At(l))
				}
			}
			sls.LogRecords().RemoveIf(func(lr plog.LogRecord) bool {
				if p.cfg.Recursive {
					return lr.Body().Type() == pcommon.ValueTypeSlice
				}
				if origLen == 0 {
					return false
				}
				origLen--
				return lr.Body().Type() == pcommon.ValueTypeSlice
			})
		}
	}
	return ld, errs
}

// setBody will set the body of the log record to the provided value
func (p *unrollProcessor) setBody(newLogRecord plog.LogRecord, expansion pcommon.Value) {
	switch expansion.Type() {
	case pcommon.ValueTypeStr:
		newLogRecord.Body().SetStr(expansion.Str())
	case pcommon.ValueTypeInt:
		newLogRecord.Body().SetInt(expansion.Int())
	case pcommon.ValueTypeDouble:
		newLogRecord.Body().SetDouble(expansion.Double())
	case pcommon.ValueTypeBool:
		newLogRecord.Body().SetBool(expansion.Bool())
	case pcommon.ValueTypeMap:
		expansion.Map().CopyTo(newLogRecord.Body().SetEmptyMap())
	case pcommon.ValueTypeSlice:
		expansion.Slice().CopyTo(newLogRecord.Body().SetEmptySlice())
	case pcommon.ValueTypeBytes:
		expansion.Bytes().CopyTo(newLogRecord.Body().SetEmptyBytes())
	case pcommon.ValueTypeEmpty:
		expansion.CopyTo(newLogRecord.Body())
	}
}
