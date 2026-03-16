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

package awssecuritylakeexporter // import "github.com/observiq/bindplane-otel-collector/exporter/awssecuritylakeexporter"

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	v100 "github.com/observiq/bindplane-otel-collector/exporter/awssecuritylakeexporter/internal/ocsf/v1_0_0"
	v110 "github.com/observiq/bindplane-otel-collector/exporter/awssecuritylakeexporter/internal/ocsf/v1_1_0"
	v120 "github.com/observiq/bindplane-otel-collector/exporter/awssecuritylakeexporter/internal/ocsf/v1_2_0"
	v130 "github.com/observiq/bindplane-otel-collector/exporter/awssecuritylakeexporter/internal/ocsf/v1_3_0"
	"github.com/parquet-go/parquet-go"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// partitionKey uniquely identifies a group of records for a single parquet file.
type partitionKey struct {
	SourceName string
	EventDay   string // YYYYMMDD
	ClassUID   int
}

// partition holds grouped records for a single parquet file.
type partition struct {
	records   []map[string]any
	schema    *parquet.Schema
	eventTime time.Time
}

// securityLakeExporter exports OCSF-formatted logs as Parquet to AWS Security Lake S3.
type securityLakeExporter struct {
	cfg    *Config
	logger *zap.Logger
	s3     S3Client

	// classToSource maps class_uid -> custom source name for routing
	classToSource map[int]string
	classToSchema map[int]*parquet.Schema
}

// newExporter creates a new Security Lake exporter.
func newExporter(cfg *Config, params exporter.Settings) (*securityLakeExporter, error) {
	schemaMap := getSchemaMap(cfg.OCSFVersion)
	classToSource := make(map[int]string, len(cfg.CustomSources))
	classToSchema := make(map[int]*parquet.Schema, len(cfg.CustomSources))
	for _, src := range cfg.CustomSources {
		classToSource[src.ClassID] = src.Name
		sm, ok := schemaMap[src.ClassID]
		if !ok {
			return nil, fmt.Errorf("no schema for class %d", src.ClassID)
		}
		schema := parquet.SchemaOf(sm)
		classToSchema[src.ClassID] = schema
	}

	return &securityLakeExporter{
		cfg:           cfg,
		logger:        params.Logger,
		classToSource: classToSource,
		classToSchema: classToSchema,
	}, nil
}

// Capabilities returns the exporter's capabilities.
func (e *securityLakeExporter) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// Start initializes the S3 client.
func (e *securityLakeExporter) Start(ctx context.Context, _ component.Host) error {
	s3, err := NewS3Client(ctx, e.cfg.Region, e.cfg.RoleARN, e.cfg.Endpoint, e.logger)
	if err != nil {
		return fmt.Errorf("creating S3 client: %w", err)
	}
	e.s3 = s3

	e.logger.Info("security lake exporter started",
		zap.String("bucket", e.cfg.S3Bucket),
		zap.String("region", e.cfg.Region),
		zap.Int("custom_sources", len(e.cfg.CustomSources)),
	)

	return nil
}

// Shutdown is a no-op; the exporterhelper handles draining the queue.
func (e *securityLakeExporter) Shutdown(_ context.Context) error {
	return nil
}

// logsDataPusher groups OCSF logs by partition, writes each group as a Parquet file, and uploads to S3.
func (e *securityLakeExporter) logsDataPusher(ctx context.Context, ld plog.Logs) error {
	partitions := make(map[partitionKey]*partition)

	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		rl := ld.ResourceLogs().At(i)
		for j := 0; j < rl.ScopeLogs().Len(); j++ {
			sl := rl.ScopeLogs().At(j)
			for k := 0; k < sl.LogRecords().Len(); k++ {
				lr := sl.LogRecords().At(k)

				if lr.Body().Type() != pcommon.ValueTypeMap {
					e.logger.Warn("skipping log record with non-map body")
					continue
				}

				body := lr.Body().Map().AsRaw()

				classUID, ok := extractClassUID(body)
				if !ok {
					e.logger.Warn("skipping log record without class_uid")
					continue
				}

				sourceName, ok := e.classToSource[classUID]
				if !ok {
					e.logger.Warn("skipping log record with unmatched class_uid",
						zap.Int("class_uid", classUID),
					)
					continue
				}

				schema := e.classToSchema[classUID]
				if schema == nil {
					return fmt.Errorf("no schema for class %d", classUID)
				}

				eventTime, err := extractEventTime(body)
				if err != nil {
					return consumererror.NewPermanent(
						fmt.Errorf("extracting event time: %w", err),
					)
				}
				eventDay := eventTime.UTC().Format("20060102")

				key := partitionKey{
					SourceName: sourceName,
					EventDay:   eventDay,
					ClassUID:   classUID,
				}

				p, ok := partitions[key]
				if !ok {
					p = &partition{
						records:   make([]map[string]any, 0, 64),
						schema:    schema,
						eventTime: eventTime,
					}
					partitions[key] = p
				}
				p.records = append(p.records, body)
			}
		}
	}

	// Write and upload each partition as a separate parquet file.
	for key, p := range partitions {
		buf, err := WriteParquet(p.schema, p.records)
		if err != nil {
			return fmt.Errorf("writing parquet: %w", err)
		}

		fileID := uuid.New().String()
		s3Key := BuildS3Key(key.SourceName, e.cfg.Region, e.cfg.AccountID, p.eventTime, fileID)

		if err := e.s3.Upload(ctx, e.cfg.S3Bucket, s3Key, bytes.NewReader(buf.Bytes())); err != nil {
			return fmt.Errorf("uploading to S3: %w", err)
		}

		e.logger.Info("uploaded parquet to S3",
			zap.String("key", s3Key),
			zap.Int("records", len(p.records)),
			zap.Int("bytes", buf.Len()),
		)
	}

	return nil
}

// extractClassUID extracts the class_uid field from an OCSF record.
func extractClassUID(record map[string]any) (int, bool) {
	v, ok := record["class_uid"]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	case int:
		return n, true
	case int32:
		return int(n), true
	default:
		return 0, false
	}
}

// extractEventTime gets the time field from an OCSF record (epoch milliseconds).
func extractEventTime(record map[string]any) (time.Time, error) {
	if t, ok := record["time"]; ok {
		switch n := t.(type) {
		case int64:
			return time.UnixMilli(n), nil
		case float64:
			return time.UnixMilli(int64(n)), nil
		case int:
			return time.UnixMilli(int64(n)), nil
		case int32:
			return time.UnixMilli(int64(n)), nil
		default:
			return time.Time{}, fmt.Errorf("invalid type for field 'time': %T", t)
		}
	}
	return time.Time{}, fmt.Errorf("missing required field 'time'")
}

// getSchemaMap returns the OCSF schema for a given OCSF version.
func getSchemaMap(version OCSFVersion) map[int]any {
	switch version {
	case OCSFVersion1_0_0:
		return v100.ClassSchemaMap
	case OCSFVersion1_1_0:
		return v110.ClassSchemaMap
	case OCSFVersion1_2_0:
		return v120.ClassSchemaMap
	case OCSFVersion1_3_0:
		return v130.ClassSchemaMap
	default:
		return nil
	}
}
