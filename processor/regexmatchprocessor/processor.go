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

package regexmatchprocessor

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/observiq/bindplane-otel-collector/processor/regexmatchprocessor/internal/matcher"
)

type regexMatchProcessor struct {
	cfg     *Config
	matcher *matcher.Matcher
}

// newRegexmatchProcessor returns a new regexmatchprocessor.
func newRegexmatchProcessor(config *Config) (*regexMatchProcessor, error) {
	matcher, err := matcher.New(config.Regexes, config.DefaultValue)
	if err != nil {
		return nil, fmt.Errorf("problem with regexes: %w", err)
	}

	return &regexMatchProcessor{
		cfg:     config,
		matcher: matcher,
	}, nil
}

// ProcessLogs implements the processor interface
func (p *regexMatchProcessor) ProcessLogs(_ context.Context, ld plog.Logs) (plog.Logs, error) {
	var errs error
	for _, rls := range ld.ResourceLogs().All() {
		for _, sls := range rls.ScopeLogs().All() {
			for _, lr := range sls.LogRecords().All() {
				if lr.Body().Type() != pcommon.ValueTypeStr {
					continue
				}
				if result := p.matcher.Match(lr.Body().Str()); result != "" {
					lr.Attributes().PutStr(p.cfg.AttributeName, result)
				}
			}
		}
	}
	return ld, errs
}
