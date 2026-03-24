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
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"
)

var testProcessorID = component.NewIDWithName(componentType, "test")

func newTestProcessorSettings() processor.Settings {
	s := processortest.NewNopSettings(componentType)
	s.ID = testProcessorID
	return s
}

func TestShutdownBeforeStart(t *testing.T) {
	dir := t.TempDir()
	ind := filepath.Join(dir, "i.txt")
	require.NoError(t, writeIndicatorFileText(ind, "x\n"))

	cfg := validBloomConfig()
	cfg.Rules[0].IndicatorFile = ind
	require.NoError(t, cfg.Validate())

	f := NewFactory()
	set := newTestProcessorSettings()
	p, err := f.CreateLogs(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	require.NotPanics(t, func() { _ = p.Shutdown(context.Background()) })
}

func TestConsumeLogs_MatchBody(t *testing.T) {
	dir := t.TempDir()
	ind := filepath.Join(dir, "i.txt")
	require.NoError(t, writeIndicatorFileText(ind, "evil.com\n"))

	cfg := validBloomConfig()
	cfg.Rules[0].IndicatorFile = ind
	require.NoError(t, cfg.Validate())

	lc := &logConsumer{ch: make(chan plog.Logs, 1)}
	f := NewFactory()
	set := newTestProcessorSettings()
	p, err := f.CreateLogs(context.Background(), set, cfg, lc)
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), nil))
	defer func() { _ = p.Shutdown(context.Background()) }()

	ld := plog.NewLogs()
	lr := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	lr.Body().SetStr("evil.com")

	require.NoError(t, p.ConsumeLogs(context.Background(), ld))
	out := <-lc.ch
	attrs := out.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes()
	v, ok := attrs.Get("threat.matched")
	require.True(t, ok)
	require.True(t, v.Bool())
	v, ok = attrs.Get("threat.rule")
	require.True(t, ok)
	require.Equal(t, "ips", v.Str())
}

func TestConsumeLogs_MatchAttribute(t *testing.T) {
	dir := t.TempDir()
	ind := filepath.Join(dir, "i.txt")
	require.NoError(t, writeIndicatorFileText(ind, "10.0.0.1\n"))

	cfg := validBloomConfig()
	cfg.Rules[0].LookupFields = []string{"client_ip"}
	cfg.Rules[0].IndicatorFile = ind
	require.NoError(t, cfg.Validate())

	lc := &logConsumer{ch: make(chan plog.Logs, 1)}
	f := NewFactory()
	set := newTestProcessorSettings()
	p, err := f.CreateLogs(context.Background(), set, cfg, lc)
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), nil))
	defer func() { _ = p.Shutdown(context.Background()) }()

	ld := plog.NewLogs()
	lr := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	lr.Body().SetStr("unrelated")
	lr.Attributes().PutStr("client_ip", "10.0.0.1")

	require.NoError(t, p.ConsumeLogs(context.Background(), ld))
	out := <-lc.ch
	attrs := out.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes()
	_, ok := attrs.Get("threat.matched")
	require.True(t, ok)
	require.Equal(t, "ips", attrs.AsRaw()["threat.rule"])
}

func TestConsumeLogs_NoMatch(t *testing.T) {
	dir := t.TempDir()
	ind := filepath.Join(dir, "i.txt")
	require.NoError(t, writeIndicatorFileText(ind, "evil.com\n"))

	cfg := validBloomConfig()
	cfg.Rules[0].IndicatorFile = ind
	require.NoError(t, cfg.Validate())

	lc := &logConsumer{ch: make(chan plog.Logs, 1)}
	f := NewFactory()
	set := newTestProcessorSettings()
	p, err := f.CreateLogs(context.Background(), set, cfg, lc)
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), nil))
	defer func() { _ = p.Shutdown(context.Background()) }()

	ld := plog.NewLogs()
	lr := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	lr.Body().SetStr("clean.example.com")

	require.NoError(t, p.ConsumeLogs(context.Background(), ld))
	out := <-lc.ch
	attrs := out.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes()
	_, ok := attrs.Get("threat.matched")
	require.False(t, ok)
}

func TestConsumeLogs_FirstMatchingRuleWins(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "1.txt")
	f2 := filepath.Join(dir, "2.txt")
	require.NoError(t, writeIndicatorFileText(f1, "same\n"))
	require.NoError(t, writeIndicatorFileText(f2, "same\n"))

	cfg := validBloomConfig()
	cfg.Rules = []Rule{
		{Name: "first", IndicatorFile: f1, LookupFields: []string{"body"}},
		{Name: "second", IndicatorFile: f2, LookupFields: []string{"body"}},
	}
	require.NoError(t, cfg.Validate())

	lc := &logConsumer{ch: make(chan plog.Logs, 1)}
	f := NewFactory()
	set := newTestProcessorSettings()
	p, err := f.CreateLogs(context.Background(), set, cfg, lc)
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), nil))
	defer func() { _ = p.Shutdown(context.Background()) }()

	ld := plog.NewLogs()
	lr := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	lr.Body().SetStr("same")

	require.NoError(t, p.ConsumeLogs(context.Background(), ld))
	out := <-lc.ch
	attrs := out.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes()
	require.Equal(t, "first", attrs.AsRaw()["threat.rule"])
}

func TestConsumeLogs_NonStringAttributeTypes(t *testing.T) {
	dir := t.TempDir()
	ind := filepath.Join(dir, "i.txt")
	require.NoError(t, writeIndicatorFileText(ind, "42\n3.14\ntrue\nrawbytes\n"))

	cfg := validBloomConfig()
	cfg.Rules[0].LookupFields = []string{"i", "f", "b", "blob"}
	cfg.Rules[0].IndicatorFile = ind
	require.NoError(t, cfg.Validate())

	lc := &logConsumer{ch: make(chan plog.Logs, 1)}
	f := NewFactory()
	set := newTestProcessorSettings()
	p, err := f.CreateLogs(context.Background(), set, cfg, lc)
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), nil))
	defer func() { _ = p.Shutdown(context.Background()) }()

	ld := plog.NewLogs()
	lr := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	lr.Body().SetStr("none")
	lr.Attributes().PutInt("i", 42)
	lr.Attributes().PutDouble("f", 3.14)
	lr.Attributes().PutBool("b", true)
	lr.Attributes().PutEmptyBytes("blob").FromRaw([]byte("rawbytes"))

	for _, k := range []string{"i", "f", "b", "blob"} {
		single := plog.NewLogs()
		slr := single.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
		slr.Body().SetStr("none")
		switch k {
		case "i":
			slr.Attributes().PutInt("i", 42)
		case "f":
			slr.Attributes().PutDouble("f", 3.14)
		case "b":
			slr.Attributes().PutBool("b", true)
		case "blob":
			slr.Attributes().PutEmptyBytes("blob").FromRaw([]byte("rawbytes"))
		}
		require.NoError(t, p.ConsumeLogs(context.Background(), single))
		out := <-lc.ch
		attrs := out.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes()
		v, ok := attrs.Get("threat.matched")
		require.True(t, ok, "key %s", k)
		require.True(t, v.Bool(), "key %s", k)
	}
}

func TestStart_MissingIndicatorFile(t *testing.T) {
	cfg := validBloomConfig()
	cfg.Rules[0].IndicatorFile = filepath.Join(t.TempDir(), "missing.txt")
	require.NoError(t, cfg.Validate())

	f := NewFactory()
	set := newTestProcessorSettings()
	p, err := f.CreateLogs(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	err = p.Start(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "indicator_file")
}

func TestStart_InvalidIndicatorJSON(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(bad, []byte(`[unclosed`), 0o644))

	cfg := validBloomConfig()
	cfg.Rules[0].IndicatorFile = bad
	require.NoError(t, cfg.Validate())

	f := NewFactory()
	set := newTestProcessorSettings()
	p, err := f.CreateLogs(context.Background(), set, cfg, consumertest.NewNop())
	require.NoError(t, err)
	err = p.Start(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse JSON array")
}

func TestConsumeLogs_JSONIndicatorFile(t *testing.T) {
	dir := t.TempDir()
	jf := filepath.Join(dir, "i.json")
	require.NoError(t, os.WriteFile(jf, []byte(`[" alpha ", " beta "]`), 0o644))

	cfg := validBloomConfig()
	cfg.Rules[0].IndicatorFile = jf
	require.NoError(t, cfg.Validate())

	lc := &logConsumer{ch: make(chan plog.Logs, 1)}
	f := NewFactory()
	set := newTestProcessorSettings()
	p, err := f.CreateLogs(context.Background(), set, cfg, lc)
	require.NoError(t, err)
	require.NoError(t, p.Start(context.Background(), nil))
	defer func() { _ = p.Shutdown(context.Background()) }()

	ld := plog.NewLogs()
	lr := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
	lr.Body().SetStr("beta")

	require.NoError(t, p.ConsumeLogs(context.Background(), ld))
	out := <-lc.ch
	attrs := out.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes()
	require.True(t, attrs.AsRaw()["threat.matched"].(bool))
}

func TestFilterKinds_Wiring(t *testing.T) {
	dir := t.TempDir()
	ind := filepath.Join(dir, "i.txt")
	require.NoError(t, writeIndicatorFileText(ind, "hit\n"))

	tests := []struct {
		name   string
		filter FilterConfig
	}{
		{
			name: "bloom",
			filter: FilterConfig{
				Kind:              "bloom",
				EstimatedCount:    500,
				FalsePositiveRate: 0.01,
			},
		},
		{
			name:   "cuckoo",
			filter: FilterConfig{Kind: "cuckoo", Capacity: 500},
		},
		{
			name:   "scalable_cuckoo",
			filter: FilterConfig{Kind: "scalable_cuckoo"},
		},
		{
			name:   "vacuum",
			filter: FilterConfig{Kind: "vacuum", Capacity: 500},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Filter: tt.filter,
				Rules: []Rule{{
					Name:          "r1",
					IndicatorFile: ind,
					LookupFields:  []string{"body"},
				}},
			}
			require.NoError(t, cfg.Validate())

			lc := &logConsumer{ch: make(chan plog.Logs, 1)}
			f := NewFactory()
			set := newTestProcessorSettings()
			p, err := f.CreateLogs(context.Background(), set, cfg, lc)
			require.NoError(t, err)
			require.NoError(t, p.Start(context.Background(), nil))
			defer func() { _ = p.Shutdown(context.Background()) }()

			ld := plog.NewLogs()
			lr := ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty()
			lr.Body().SetStr("hit")
			require.NoError(t, p.ConsumeLogs(context.Background(), ld))
			out := <-lc.ch
			attrs := out.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes()
			matched, ok := attrs.Get("threat.matched")
			require.True(t, ok)
			require.True(t, matched.Bool())
		})
	}
}

func TestPcommonValueToString_MapAndSlice(t *testing.T) {
	m := pcommon.NewValueMap()
	require.Equal(t, "{}", pcommonValueToString(m))

	sl := pcommon.NewValueSlice()
	got := pcommonValueToString(sl)
	require.True(t, got == "" || got == "[]", "slice as string: %q", got)
}

type logConsumer struct {
	ch chan plog.Logs
}

func (l *logConsumer) ConsumeLogs(_ context.Context, ld plog.Logs) error {
	if l.ch != nil {
		l.ch <- ld
	}
	return nil
}

func (l *logConsumer) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}
