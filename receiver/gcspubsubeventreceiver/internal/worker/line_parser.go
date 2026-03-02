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
	"context"
	"fmt"
	"io"
	"iter"

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
func (p *lineParser) Parse(_ context.Context, startOffset int64) (logs iter.Seq2[any, error], err error) {
	// skip to the start offset
	_, err = io.CopyN(io.Discard, p.reader, startOffset)
	if err != nil {
		return nil, fmt.Errorf("discard to offset: %w", err)
	}

	// logs is a sequence of log records that can be used with the provided appender
	return func(yield func(any, error) bool) {
		for {
			lineBytes, _, err := p.reader.ReadLine()
			if err != nil {
				if err == io.EOF {
					return
				}
				if !yield(nil, err) {
					return
				}
			}

			// only yield non-empty lines
			if len(lineBytes) > 0 {
				if !yield(string(lineBytes), nil) {
					return
				}
			}
		}
	}, nil
}

// AppendLogBody appends the log record to the log record body using SetStr.
func (p *lineParser) AppendLogBody(_ context.Context, lr plog.LogRecord, record any) error {
	str, ok := record.(string)
	if !ok {
		return fmt.Errorf("expected string record, got %T", record)
	}
	lr.Body().SetStr(str)
	return nil
}
