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

package awssecuritylakeexporter

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap/zaptest"
)

// mockS3Client records uploads for assertion.
type mockS3Client struct {
	uploads []mockUpload
	err     error
}

type mockUpload struct {
	bucket string
	key    string
	body   []byte
}

func (m *mockS3Client) Upload(_ context.Context, bucket, key string, body io.Reader) error {
	if m.err != nil {
		return m.err
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	m.uploads = append(m.uploads, mockUpload{bucket: bucket, key: key, body: data})
	return nil
}

// newTestExporter creates an exporter with a mock S3 client for testing.
func newTestExporter(t *testing.T, classIDs []int, s3 S3Client) *securityLakeExporter {
	t.Helper()

	cfg := validConfig()
	cfg.CustomSources = make([]SecurityLakeCustomSource, len(classIDs))
	for i, id := range classIDs {
		cfg.CustomSources[i] = SecurityLakeCustomSource{
			Name:    fmt.Sprintf("source-%d", id),
			ClassID: id,
		}
	}

	schemaMap := getSchemaMap(cfg.OCSFVersion)
	classToSource := make(map[int]string, len(cfg.CustomSources))
	classToSchema := make(map[int]*parquet.Schema, len(cfg.CustomSources))
	for _, src := range cfg.CustomSources {
		classToSource[src.ClassID] = src.Name
		classToSchema[src.ClassID] = parquet.SchemaOf(schemaMap[src.ClassID])
	}

	return &securityLakeExporter{
		cfg:           cfg,
		logger:        zaptest.NewLogger(t),
		s3:            s3,
		classToSource: classToSource,
		classToSchema: classToSchema,
	}
}

// makeLog creates a plog.Logs with a single log record from the given body map.
func makeLog(body map[string]any) plog.Logs {
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	lr := sl.LogRecords().AppendEmpty()
	if err := lr.Body().SetEmptyMap().FromRaw(body); err != nil {
		panic(err)
	}
	return ld
}

// makeLogs creates a plog.Logs with multiple log records from the given body maps.
func makeLogs(bodies []map[string]any) plog.Logs {
	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()
	for _, body := range bodies {
		lr := sl.LogRecords().AppendEmpty()
		if err := lr.Body().SetEmptyMap().FromRaw(body); err != nil {
			panic(err)
		}
	}
	return ld
}

func ocsf(classUID int, timeMs int64) map[string]any {
	return map[string]any{
		"class_uid":    int64(classUID),
		"time":         timeMs,
		"activity_id":  int64(1),
		"category_uid": int64(3),
		"severity_id":  int64(1),
		"type_uid":     int64(300201),
		"metadata": map[string]any{
			"version": "1.3.0",
			"product": map[string]any{
				"vendor_name": "Test",
				"name":        "TestProduct",
			},
		},
	}
}

func TestCapabilities(t *testing.T) {
	mock := &mockS3Client{}
	exp := newTestExporter(t, []int{3002}, mock)
	caps := exp.Capabilities()
	assert.False(t, caps.MutatesData)
}

func TestShutdown(t *testing.T) {
	mock := &mockS3Client{}
	exp := newTestExporter(t, []int{3002}, mock)
	err := exp.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestLogsDataPusher_SingleRecord(t *testing.T) {
	mock := &mockS3Client{}
	exp := newTestExporter(t, []int{3002}, mock)

	ld := makeLog(ocsf(3002, 1700000000000))
	err := exp.logsDataPusher(context.Background(), ld)
	require.NoError(t, err)

	require.Len(t, mock.uploads, 1)
	assert.Equal(t, exp.cfg.S3Bucket, mock.uploads[0].bucket)
	assert.Contains(t, mock.uploads[0].key, "source-3002")
	assert.Contains(t, mock.uploads[0].key, ".parquet")
	assert.True(t, len(mock.uploads[0].body) > 0)
}

func TestLogsDataPusher_MultipleRecordsSamePartition(t *testing.T) {
	mock := &mockS3Client{}
	exp := newTestExporter(t, []int{3002}, mock)

	bodies := []map[string]any{
		ocsf(3002, 1700000000000),
		ocsf(3002, 1700000001000),
		ocsf(3002, 1700000002000),
	}
	ld := makeLogs(bodies)
	err := exp.logsDataPusher(context.Background(), ld)
	require.NoError(t, err)

	// Same class_uid and same day -> single partition -> single upload.
	require.Len(t, mock.uploads, 1)
}

func TestLogsDataPusher_MultiplePartitions(t *testing.T) {
	mock := &mockS3Client{}
	// 3002 = Authentication, 3001 = AccountChange
	exp := newTestExporter(t, []int{3002, 3001}, mock)

	bodies := []map[string]any{
		ocsf(3002, 1700000000000),
		ocsf(3001, 1700000000000),
	}
	ld := makeLogs(bodies)
	err := exp.logsDataPusher(context.Background(), ld)
	require.NoError(t, err)

	// Different class_uids -> two partitions -> two uploads.
	require.Len(t, mock.uploads, 2)
}

func TestLogsDataPusher_DifferentDays(t *testing.T) {
	mock := &mockS3Client{}
	exp := newTestExporter(t, []int{3002}, mock)

	day1 := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	day2 := time.Date(2024, 1, 16, 12, 0, 0, 0, time.UTC).UnixMilli()

	bodies := []map[string]any{
		ocsf(3002, day1),
		ocsf(3002, day2),
	}
	ld := makeLogs(bodies)
	err := exp.logsDataPusher(context.Background(), ld)
	require.NoError(t, err)

	// Same class but different days -> two partitions.
	require.Len(t, mock.uploads, 2)
}

func TestLogsDataPusher_SkipsNonMapBody(t *testing.T) {
	mock := &mockS3Client{}
	exp := newTestExporter(t, []int{3002}, mock)

	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()

	// String body (non-map) — should be skipped.
	lr := sl.LogRecords().AppendEmpty()
	lr.Body().SetStr("plain text log")

	err := exp.logsDataPusher(context.Background(), ld)
	require.NoError(t, err)
	assert.Empty(t, mock.uploads)
}

func TestLogsDataPusher_SkipsMissingClassUID(t *testing.T) {
	mock := &mockS3Client{}
	exp := newTestExporter(t, []int{3002}, mock)

	body := map[string]any{
		"time": int64(1700000000000),
		// No class_uid.
	}
	ld := makeLog(body)
	err := exp.logsDataPusher(context.Background(), ld)
	require.NoError(t, err)
	assert.Empty(t, mock.uploads)
}

func TestLogsDataPusher_SkipsUnmatchedClassUID(t *testing.T) {
	mock := &mockS3Client{}
	exp := newTestExporter(t, []int{3002}, mock)

	// 9999 is not in the custom sources.
	body := ocsf(9999, 1700000000000)
	ld := makeLog(body)
	err := exp.logsDataPusher(context.Background(), ld)
	require.NoError(t, err)
	assert.Empty(t, mock.uploads)
}

func TestLogsDataPusher_ErrorOnMissingTime(t *testing.T) {
	mock := &mockS3Client{}
	exp := newTestExporter(t, []int{3002}, mock)

	body := map[string]any{
		"class_uid": int64(3002),
		// No time field.
	}
	ld := makeLog(body)
	err := exp.logsDataPusher(context.Background(), ld)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "extracting event time")
}

func TestLogsDataPusher_ErrorOnInvalidTimeType(t *testing.T) {
	mock := &mockS3Client{}
	exp := newTestExporter(t, []int{3002}, mock)

	body := map[string]any{
		"class_uid": int64(3002),
		"time":      "not-a-number",
	}
	ld := makeLog(body)
	err := exp.logsDataPusher(context.Background(), ld)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "extracting event time")
}

func TestLogsDataPusher_S3UploadError(t *testing.T) {
	mock := &mockS3Client{err: fmt.Errorf("access denied")}
	exp := newTestExporter(t, []int{3002}, mock)

	ld := makeLog(ocsf(3002, 1700000000000))
	err := exp.logsDataPusher(context.Background(), ld)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "uploading to S3")
}

func TestLogsDataPusher_EmptyLogs(t *testing.T) {
	mock := &mockS3Client{}
	exp := newTestExporter(t, []int{3002}, mock)

	ld := plog.NewLogs()
	err := exp.logsDataPusher(context.Background(), ld)
	require.NoError(t, err)
	assert.Empty(t, mock.uploads)
}

func TestLogsDataPusher_S3KeyFormat(t *testing.T) {
	mock := &mockS3Client{}
	exp := newTestExporter(t, []int{3002}, mock)

	ld := makeLog(ocsf(3002, 1700000000000))
	err := exp.logsDataPusher(context.Background(), ld)
	require.NoError(t, err)

	require.Len(t, mock.uploads, 1)
	key := mock.uploads[0].key
	assert.True(t, strings.HasPrefix(key, "ext/source-3002/region=us-east-1/accountId=123456789012/eventDay="))
	assert.True(t, strings.HasSuffix(key, ".parquet"))
}

func TestLogsDataPusher_MixedValidAndSkipped(t *testing.T) {
	mock := &mockS3Client{}
	exp := newTestExporter(t, []int{3002}, mock)

	ld := plog.NewLogs()
	rl := ld.ResourceLogs().AppendEmpty()
	sl := rl.ScopeLogs().AppendEmpty()

	// Valid record.
	lr1 := sl.LogRecords().AppendEmpty()
	require.NoError(t, lr1.Body().SetEmptyMap().FromRaw(ocsf(3002, 1700000000000)))

	// Non-map body — skipped.
	lr2 := sl.LogRecords().AppendEmpty()
	lr2.Body().SetStr("plain text")

	// Missing class_uid — skipped.
	lr3 := sl.LogRecords().AppendEmpty()
	require.NoError(t, lr3.Body().SetEmptyMap().FromRaw(map[string]any{"time": int64(1700000000000)}))

	// Unmatched class_uid — skipped.
	lr4 := sl.LogRecords().AppendEmpty()
	require.NoError(t, lr4.Body().SetEmptyMap().FromRaw(ocsf(9999, 1700000000000)))

	err := exp.logsDataPusher(context.Background(), ld)
	require.NoError(t, err)

	// Only the first valid record should produce an upload.
	require.Len(t, mock.uploads, 1)
}

func TestExtractClassUID(t *testing.T) {
	tests := []struct {
		name   string
		record map[string]any
		want   int
		wantOK bool
	}{
		{"int64", map[string]any{"class_uid": int64(3002)}, 3002, true},
		{"float64", map[string]any{"class_uid": float64(3002)}, 3002, true},
		{"int", map[string]any{"class_uid": 3002}, 3002, true},
		{"int32", map[string]any{"class_uid": int32(3002)}, 3002, true},
		{"string", map[string]any{"class_uid": "3002"}, 0, false},
		{"missing", map[string]any{}, 0, false},
		{"nil", map[string]any{"class_uid": nil}, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractClassUID(tt.record)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractEventTime(t *testing.T) {
	tests := []struct {
		name    string
		record  map[string]any
		want    time.Time
		wantErr string
	}{
		{
			name:   "valid",
			record: map[string]any{"time": int64(1700000000000)},
			want:   time.UnixMilli(1700000000000),
		},
		{
			name:    "missing",
			record:  map[string]any{},
			wantErr: "missing required field 'time'",
		},
		{
			name:    "wrong type string",
			record:  map[string]any{"time": "2024-01-01"},
			wantErr: "invalid type for field 'time'",
		},
		{
			name:   "float64",
			record: map[string]any{"time": float64(1700000000000)},
			want:   time.UnixMilli(1700000000000),
		},
		{
			name:   "int",
			record: map[string]any{"time": int(1700000000000)},
			want:   time.UnixMilli(1700000000000),
		},
		{
			name:   "int32",
			record: map[string]any{"time": int32(1700000000)},
			want:   time.UnixMilli(1700000000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractEventTime(tt.record)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGetSchemaMap(t *testing.T) {
	tests := []struct {
		version OCSFVersion
		wantNil bool
	}{
		{OCSFVersion1_0_0, false},
		{OCSFVersion1_1_0, false},
		{OCSFVersion1_2_0, false},
		{OCSFVersion1_3_0, false},
		{"9.9.9", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.version), func(t *testing.T) {
			m := getSchemaMap(tt.version)
			if tt.wantNil {
				assert.Nil(t, m)
			} else {
				assert.NotNil(t, m)
				assert.True(t, len(m) > 0)
			}
		})
	}
}

func TestBuildS3Key(t *testing.T) {
	eventTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
	key := BuildS3Key("my-source", "us-east-1", "123456789012", eventTime, "abc-123")
	assert.Equal(t, "ext/my-source/region=us-east-1/accountId=123456789012/eventDay=20240115/my-source_1705329000_abc-123.parquet", key)
}
