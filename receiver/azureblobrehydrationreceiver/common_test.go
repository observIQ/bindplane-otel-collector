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

package azureblobrehydrationreceiver //import "github.com/observiq/bindplane-agent/receiver/azureblobrehydrationreceiver"

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// File contains test helper functions

// generateTestMetrics generates test metrics and the marshaled json output bytes
func generateTestMetrics(t *testing.T) (md pmetric.Metrics, jsonBytes []byte) {
	t.Helper()

	md = pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	sm := rm.ScopeMetrics().AppendEmpty()
	m := sm.Metrics().AppendEmpty()
	m.SetName("metric")
	gm := m.SetEmptyGauge()
	dp := gm.DataPoints().AppendEmpty()
	dp.SetIntValue(1)
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	marshaler := &pmetric.JSONMarshaler{}

	var err error
	jsonBytes, err = marshaler.MarshalMetrics(md)
	require.NoError(t, err)

	return md, jsonBytes
}

// generateTestLogs generates test logs and the marshaled json output bytes
func generateTestLogs(t *testing.T) (ld plog.Logs, jsonBytes []byte) {
	t.Helper()

	ld = plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	l := sl.LogRecords().AppendEmpty()
	l.Body().SetStr("body")
	l.Attributes().PutBool("bool", true)
	l.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	marshaler := &plog.JSONMarshaler{}

	var err error
	jsonBytes, err = marshaler.MarshalLogs(ld)
	require.NoError(t, err)

	return ld, jsonBytes
}

// generateTestTraces generates test traces and the marshaled json output bytes
func generateTestTraces(t *testing.T) (td ptrace.Traces, jsonBytes []byte) {
	t.Helper()

	td = ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()
	s := ss.Spans().AppendEmpty()
	s.Attributes().PutBool("bool", true)
	s.SetName("span")
	s.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	marshaler := &ptrace.JSONMarshaler{}

	var err error
	jsonBytes, err = marshaler.MarshalTraces(td)
	require.NoError(t, err)

	return td, jsonBytes
}
