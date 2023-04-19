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

package maskprocessor

import (
	"context"
	"fmt"
	"regexp"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

const (
	resourceField   = "resource"
	attributesField = "attributes"
	bodyField       = "body"
)

var defaultRules = map[string]*regexp.Regexp{
	"email":       regexp.MustCompile(`\b[a-z0-9._%\+\-—|]+@[a-z0-9.\-—|]+\.[a-z|]{2,6}\b`),
	"ssn":         regexp.MustCompile(`\b\d{3}[- ]\d{2}[- ]\d{4}\b`),
	"credit_card": regexp.MustCompile(`\b(?:(?:(?:\d{4}[- ]?){3}\d{4}|\d{15,16}))\b`),
	"phone":       regexp.MustCompile(`\b((\+|\b)[1l][\-\. ])?\(?\b[\dOlZSB]{3,5}([\-\. ]|\) ?)[\dOlZSB]{3}[\-\. ][\dOlZSB]{4}\b`),
	"ipv4":        regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`),
}

// maskProcessor is the processor used to mask data.
type maskProcessor struct {
	logger           *zap.Logger
	cfg              *Config
	rules            map[string]*regexp.Regexp
	maskResourceFunc func(k string, v pcommon.Value) bool
	maskAttrsFunc    func(k string, v pcommon.Value) bool
}

// newProcessor creates a new mask processor.
func newProcessor(logger *zap.Logger, cfg *Config) *maskProcessor {
	return &maskProcessor{
		logger: logger,
		cfg:    cfg,
	}
}

// start is used to start the processor.
func (p *maskProcessor) start(context.Context, component.Host) error {
	rules, err := p.createRules()
	if err != nil {
		return err
	}

	p.rules = rules
	p.maskResourceFunc = p.createMaskFunc(resourceField)
	p.maskAttrsFunc = p.createMaskFunc(attributesField)
	return nil
}

// processLogs masks incoming logs.
func (p *maskProcessor) processLogs(_ context.Context, ld plog.Logs) (plog.Logs, error) {
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resource := ld.ResourceLogs().At(i)
		resource.Resource().Attributes().Range(p.maskResourceFunc)
		for j := 0; j < resource.ScopeLogs().Len(); j++ {
			scope := resource.ScopeLogs().At(j)
			for k := 0; k < scope.LogRecords().Len(); k++ {
				logs := scope.LogRecords().At(k)
				logs.Attributes().Range(p.maskAttrsFunc)
				p.maskValue(bodyField, logs.Body())
			}
		}
	}

	return ld, nil
}

// processTraces masks incoming traces.
func (p *maskProcessor) processTraces(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		resource := td.ResourceSpans().At(i)
		resource.Resource().Attributes().Range(p.maskResourceFunc)
		for j := 0; j < resource.ScopeSpans().Len(); j++ {
			scope := resource.ScopeSpans().At(j)
			for k := 0; k < scope.Spans().Len(); k++ {
				spans := scope.Spans().At(k)
				spans.Attributes().Range(p.maskAttrsFunc)
			}
		}
	}

	return td, nil
}

// processMetrics masks incoming metrics.
func (p *maskProcessor) processMetrics(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		resource := md.ResourceMetrics().At(i)
		resource.Resource().Attributes().Range(p.maskResourceFunc)
		for j := 0; j < resource.ScopeMetrics().Len(); j++ {
			scope := resource.ScopeMetrics().At(j)
			for k := 0; k < scope.Metrics().Len(); k++ {
				metrics := scope.Metrics().At(k)
				switch metrics.Type() {
				case pmetric.MetricTypeSum:
					p.processSum(metrics.Sum())
				case pmetric.MetricTypeGauge:
					p.processGauge(metrics.Gauge())
				case pmetric.MetricTypeSummary:
					p.processSummary(metrics.Summary())
				case pmetric.MetricTypeHistogram:
					p.processHistogram(metrics.Histogram())
				case pmetric.MetricTypeExponentialHistogram:
					p.processExponentialHistogram(metrics.ExponentialHistogram())
				}
			}
		}
	}

	return md, nil
}

// processSum masks a sum metric.
func (p *maskProcessor) processSum(sum pmetric.Sum) {
	for i := 0; i < sum.DataPoints().Len(); i++ {
		sum.DataPoints().At(i).Attributes().Range(p.maskAttrsFunc)
	}
}

// processGauge masks a gauge metric.
func (p *maskProcessor) processGauge(gauge pmetric.Gauge) {
	for i := 0; i < gauge.DataPoints().Len(); i++ {
		gauge.DataPoints().At(i).Attributes().Range(p.maskAttrsFunc)
	}
}

// processSummary masks a summary metric.
func (p *maskProcessor) processSummary(summary pmetric.Summary) {
	for i := 0; i < summary.DataPoints().Len(); i++ {
		summary.DataPoints().At(i).Attributes().Range(p.maskAttrsFunc)
	}
}

// processHistogram masks a histogram metric.
func (p *maskProcessor) processHistogram(histogram pmetric.Histogram) {
	for i := 0; i < histogram.DataPoints().Len(); i++ {
		histogram.DataPoints().At(i).Attributes().Range(p.maskAttrsFunc)
	}
}

// processExponentialHistogram masks a histogram metric.
func (p *maskProcessor) processExponentialHistogram(histogram pmetric.ExponentialHistogram) {
	for i := 0; i < histogram.DataPoints().Len(); i++ {
		histogram.DataPoints().At(i).Attributes().Range(p.maskAttrsFunc)
	}
}

// maskValue masks a pcommon.Value.
func (p *maskProcessor) maskValue(field string, value pcommon.Value) {
	for _, excludeField := range p.cfg.Exclude {
		if field == excludeField {
			return
		}
	}

	switch value.Type() {
	case pcommon.ValueTypeMap:
		maskFunc := p.createMaskFunc(field)
		value.Map().Range(maskFunc)
	case pcommon.ValueTypeStr:
		p.maskString(value)
	case pcommon.ValueTypeSlice:
		// Search for strings in a slice and apply mask
		for i := 0; i < value.Slice().Len(); i++ {
			sliceVal := value.Slice().At(i)
			if sliceVal.Type() == pcommon.ValueTypeStr {
				p.maskString(sliceVal)
			}
		}
	}
}

// maskString masks a pcommon string.
func (p *maskProcessor) maskString(value pcommon.Value) {
	strValue := value.Str()

	for mask, rule := range p.rules {
		if !rule.MatchString(strValue) {
			continue
		}

		strValue = rule.ReplaceAllString(strValue, mask)
	}

	if strValue != value.Str() {
		value.SetStr(strValue)
	}
}

// createMaskFunc creates a func for ranging through a pcommon.Map and masking its values.
func (p *maskProcessor) createMaskFunc(field string) func(k string, v pcommon.Value) bool {
	return func(k string, v pcommon.Value) bool {
		childField := fmt.Sprintf("%s.%s", field, k)
		p.maskValue(childField, v)
		return true
	}
}

// createRules creates a map of rules for the processor.
func (p *maskProcessor) createRules() (map[string]*regexp.Regexp, error) {
	rules := defaultRules
	if len(p.cfg.Rules) > 0 {
		var err error
		rules, err = compileRules(p.cfg.Rules)
		if err != nil {
			return nil, err
		}
	}

	formattedRules := formatRuleMaskName(rules)
	return formattedRules, nil
}

// compileRules compiles rules from the provided map of expressions.
func compileRules(exprs map[string]string) (map[string]*regexp.Regexp, error) {
	rules := make(map[string]*regexp.Regexp)
	for key, expr := range exprs {
		rule, err := regexp.Compile(expr)
		if err != nil {
			return nil, fmt.Errorf("rule '%s' does not compile as valid regex", key)
		}

		rules[key] = rule
	}
	return rules, nil
}

func formatRuleMaskName(rules map[string]*regexp.Regexp) map[string]*regexp.Regexp {
	formattedRules := make(map[string]*regexp.Regexp, len(rules))
	for ruleName, rule := range rules {
		maskedName := fmt.Sprintf("[masked_%s]", ruleName)
		formattedRules[maskedName] = rule
	}

	return formattedRules
}
