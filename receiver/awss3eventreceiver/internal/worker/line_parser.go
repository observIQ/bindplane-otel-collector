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

package worker

import (
	"fmt"
	"io"
	"iter"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

type lineParser struct {
	reader BufferedReader
}

// NewLineParser creates a new line parser.
func NewLineParser(reader BufferedReader) LogParser {
	return &lineParser{
		reader: reader,
	}
}

func (p *lineParser) Offset() int64 {
	return p.reader.Offset()
}

// Parse parses the log records from the reader using ReadLine.
func (p *lineParser) Parse(record events.S3EventRecord, maxLogsEmitted int, startOffset int64) (logs iter.Seq2[plog.Logs, error], err error) {
	// skip to the start offset
	_, err = io.CopyN(io.Discard, p.reader, startOffset)
	if err != nil {
		return nil, fmt.Errorf("discard to offset: %w", err)
	}

	now := time.Now()
	bucket := record.S3.Bucket.Name
	key := record.S3.Object.Key

	ld := plog.NewLogs()
	rls := ld.ResourceLogs().AppendEmpty()
	rls.Resource().Attributes().PutStr("aws.s3.bucket", bucket)
	rls.Resource().Attributes().PutStr("aws.s3.key", key)
	lrs := rls.ScopeLogs().AppendEmpty().LogRecords()

	// logs is a sequence of log records that can be used with the provided appender
	return func(yield func(plog.Logs, error) bool) {
		for {
			lineBytes, _, err := p.reader.ReadLine()
			if err != nil {
				if err == io.EOF {
					break
				}
				if !yield(plog.NewLogs(), err) {
					return
				}
			}

			// only yield non-empty lines
			if len(lineBytes) > 0 {
				// Create a log record for this line fragment
				lr := lrs.AppendEmpty()
				lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(now))
				lr.SetTimestamp(pcommon.NewTimestampFromTime(record.EventTime))
				lr.Body().SetStr(string(lineBytes))
			}

			if ld.LogRecordCount() >= maxLogsEmitted {
				if !yield(ld, nil) {
					return
				}
				ld = plog.NewLogs()
				rls = ld.ResourceLogs().AppendEmpty()
				rls.Resource().Attributes().PutStr("aws.s3.bucket", bucket)
				rls.Resource().Attributes().PutStr("aws.s3.key", key)
				lrs = rls.ScopeLogs().AppendEmpty().LogRecords()
			}
		}

		if ld.LogRecordCount() > 0 {
			if !yield(ld, nil) {
				return
			}
		}
	}, nil
}
