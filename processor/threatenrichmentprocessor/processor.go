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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/observiq/bindplane-otel-collector/internal/filter"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/stanza/adapter"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension/xextension/storage"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"
)

// ruleState holds a loaded filter and the fields to check for one rule.
type ruleState struct {
	name         string
	f            filter.Filter
	lookupFields []string
}

const bodyFieldKey = "body"

type threatEnrichmentProcessor struct {
	logger        *zap.Logger
	cfg           *Config
	id            component.ID
	storageClient storage.Client
	rules         []ruleState
}

func newProcessor(set processor.Settings, cfg *Config) *threatEnrichmentProcessor {
	return &threatEnrichmentProcessor{
		logger:        set.Logger,
		cfg:           cfg,
		id:            set.ID,
		storageClient: storage.NewNopClient(),
	}
}

func (p *threatEnrichmentProcessor) start(ctx context.Context, host component.Host) error {
	if p.cfg.StorageID != nil {
		storageClient, err := adapter.GetStorageClient(ctx, host, p.cfg.StorageID, p.id)
		if err != nil {
			return fmt.Errorf("failed to get storage client: %w", err)
		}
		p.storageClient = storageClient
		p.logger.Info("storage extension enabled", zap.String("storage_id", p.cfg.StorageID.String()))
	}

	p.rules = make([]ruleState, 0, len(p.cfg.Rules))
	for _, r := range p.cfg.Rules {
		filterCfg := &p.cfg.Filter
		if r.Filter != nil {
			filterCfg = r.Filter
		}
		opts, err := filterCfg.toFilterConfig()
		if err != nil {
			return fmt.Errorf("rule %q filter config: %w", r.Name, err)
		}
		f, err := filter.NewFilterFromConfig(opts)
		if err != nil {
			return fmt.Errorf("rule %q create filter: %w", r.Name, err)
		}
		values, err := loadIndicatorFile(r.IndicatorFile)
		if err != nil {
			return fmt.Errorf("rule %q indicator_file: %w", r.Name, err)
		}
		for _, v := range values {
			f.AddString(v)
		}
		p.logger.Info("loaded rule", zap.String("rule", r.Name), zap.String("indicator_file", r.IndicatorFile), zap.Int("count", len(values)), zap.Strings("lookup_fields", r.LookupFields))
		p.rules = append(p.rules, ruleState{name: r.Name, f: f, lookupFields: r.LookupFields})
	}

	p.logger.Info("threat enrichment processor started", zap.Int("rules", len(p.rules)))
	return nil
}

func (p *threatEnrichmentProcessor) shutdown(ctx context.Context) error {
	p.logger.Info("threat enrichment processor shutting down")

	if p.storageClient != nil {
		if err := p.storageClient.Close(ctx); err != nil {
			return fmt.Errorf("failed to close storage client: %w", err)
		}
	}

	return nil
}

func (p *threatEnrichmentProcessor) processLogs(_ context.Context, ld plog.Logs) (plog.Logs, error) {
	if len(p.rules) == 0 {
		return ld, nil
	}
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resourceLog := ld.ResourceLogs().At(i)
		for j := 0; j < resourceLog.ScopeLogs().Len(); j++ {
			scopeLog := resourceLog.ScopeLogs().At(j)
			for k := 0; k < scopeLog.LogRecords().Len(); k++ {
				lr := scopeLog.LogRecords().At(k)
				p.checkLogRecord(lr)
			}
		}
	}
	return ld, nil
}

// checkLogRecord checks each rule: for each lookup field, gets the value from the log, looks it up in the rule's filter.
// If found, enriches the record with threat.matched=true and threat.rule=<rule name> and stops checking further rules for this log.
func (p *threatEnrichmentProcessor) checkLogRecord(lr plog.LogRecord) {
	for _, rule := range p.rules {
		for _, field := range rule.lookupFields {
			val := p.getLookupValue(lr, field)
			if val != "" {
				p.logger.Debug("checking value against filter",
					zap.String("rule", rule.name),
					zap.String("field", field),
					zap.Int("value_len", len(val)),
				)
			}
			if val == "" {
				continue
			}
			if rule.f.MayContainString(val) {
				p.logger.Debug("threat indicator matched",
					zap.String("rule", rule.name),
					zap.String("field", field),
					zap.Int("value_len", len(val)),
				)
				lr.Attributes().PutBool("threat.matched", true)
				lr.Attributes().PutStr("threat.rule", rule.name)
				return
			}
		}
	}
}

// getLookupValue returns the string value from the log for the given field.
// If field is "body", returns the log body as string; otherwise returns the attribute value for that key.
func (p *threatEnrichmentProcessor) getLookupValue(lr plog.LogRecord, field string) string {
	if field == bodyFieldKey {
		body := lr.Body()
		if body.Type() != pcommon.ValueTypeStr {
			return ""
		}
		return strings.TrimSpace(body.Str())
	}
	v, ok := lr.Attributes().Get(field)
	if !ok {
		return ""
	}
	return strings.TrimSpace(pcommonValueToString(v))
}

func pcommonValueToString(v pcommon.Value) string {
	switch v.Type() {
	case pcommon.ValueTypeStr:
		return v.Str()
	case pcommon.ValueTypeInt:
		return v.AsString()
	case pcommon.ValueTypeDouble:
		return v.AsString()
	case pcommon.ValueTypeBool:
		return v.AsString()
	case pcommon.ValueTypeBytes:
		return string(v.Bytes().AsRaw())
	default:
		return v.AsString()
	}
}

// loadIndicatorFile reads indicator values from path. Supports: plain text (one value per line) or JSON array of strings.
func loadIndicatorFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var arr []string
		if err := json.Unmarshal(data, &arr); err != nil {
			return nil, fmt.Errorf("parse JSON array: %w", err)
		}
		out := make([]string, 0, len(arr))
		for _, s := range arr {
			t := strings.TrimSpace(s)
			if t != "" {
				out = append(out, t)
			}
		}
		return out, nil
	}
	var out []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		s := strings.TrimSpace(scanner.Text())
		if s != "" {
			out = append(out, s)
		}
	}
	return out, scanner.Err()
}
