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
	"iter"
	"slices"

	"github.com/linkedin/goavro/v2"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

const (
	avroOcfMagicString = "Obj\x01"
)

var (
	avroOcfMagicBytes = []byte(avroOcfMagicString)
)

type avroOcfParser struct {
	reader  BufferedReader
	logger  *zap.Logger
	counter int64
}

var _ LogParser = (*avroOcfParser)(nil)

// NewAvroOcfParser creates a new Avro OCF parser. Before attempting to parse the stream,
// call StartsWithAvroOcfMagic to check if the stream starts with the Avro OCF magic
// string.
func NewAvroOcfParser(reader BufferedReader, logger *zap.Logger) LogParser {
	return &avroOcfParser{
		reader:  reader,
		logger:  logger,
		counter: 0,
	}
}

// StartsWithAvroOcfMagic checks if the reader starts with the Avro OCF magic string.
func StartsWithAvroOcfMagic(reader BufferedReader) (bool, error) {
	bytes, err := reader.Peek(len(avroOcfMagicBytes))
	if err != nil {
		return false, err
	}
	return slices.Equal(bytes, avroOcfMagicBytes), nil
}

// Parse parses the avro ocf stream into a sequence of log records. An avro ocf stream
// starts with the schema and then contains blocks of records. The parser will return a
// sequence of records from the ocf reader or an error if a new ocf reader cannot be
// created. The parser will skip the records before the startOffset and expects an offset
// in the number of records read so far.
func (p *avroOcfParser) Parse(_ context.Context, startOffset int64) (logs iter.Seq2[any, error], err error) {
	ocfReader, err := goavro.NewOCFReader(p.reader)
	if err != nil {
		return nil, fmt.Errorf("create ocf reader: %w", err)
	}

	// yield a sequence of records from the ocf reader
	return func(yield func(any, error) bool) {
		for ocfReader.Scan() {
			record, err := ocfReader.Read()

			if err != nil {
				currentOffset := p.Offset()
				p.counter += ocfReader.RemainingBlockItems()

				// if read fails, yield the error and skip the block
				if !yield(nil, err) {
					return
				}
				p.logger.Error("avro ocf read error, skipping block",
					zap.Error(err),
					zap.Int64("offset", currentOffset),
					zap.Int64("new_offset", p.Offset()),
					zap.Int64("remaining_block_items", ocfReader.RemainingBlockItems()),
				)
				ocfReader.SkipThisBlockAndReset()
				continue
			}

			// skip if we are still before the startOffset
			p.counter++
			if p.counter <= startOffset {
				continue
			}

			// yield the avro record
			if !yield(record, err) {
				return
			}
		}
	}, nil
}

// Offset returns the number of records read so far. We use a counter instead of an offset
// in the reader because the avro library will read an entire block of records at a time,
// and we want to track the number of records read so far so that we can skip the records
// before the startOffset.
func (p *avroOcfParser) Offset() int64 {
	return p.counter
}

// AppendLogBody appends the avro record to the log record.
func (p *avroOcfParser) AppendLogBody(_ context.Context, lr plog.LogRecord, record any) error {
	return lr.Body().FromRaw(record)
}
