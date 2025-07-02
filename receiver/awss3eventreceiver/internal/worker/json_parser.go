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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"

	"go.opentelemetry.io/collector/pdata/plog"
)

var (
	// ErrNotArrayOrKnownObject is returned when the JSON stream is not a valid array or object
	// with a known key. When this occurs, try to parse as text.
	ErrNotArrayOrKnownObject = errors.New("expected array or object with known key")
)

const (
	// maxRecordsSearchBytes is the maximum number of bytes to search for a "Records" key in
	// the first 4096 bytes of the JSON stream. This is to avoid parsing the entire file
	// looking for a "Records" key and not finding it.
	maxRecordsSearchBytes = 4096
)

type jsonParser struct {
	reader  BufferedReader
	decoder *json.Decoder
}

var _ LogParser = (*jsonParser)(nil)

// NewJSONParser creates a new JSON parser.
func NewJSONParser(reader BufferedReader) LogParser {
	return &jsonParser{
		reader:  reader,
		decoder: json.NewDecoder(reader),
	}
}

// StartsWithJSONObjectOrArray returns true if the reader starts with a JSON object or
// array, allowing some space before the starting delimiter. It uses Peek and will not
// move the reader.
func StartsWithJSONObjectOrArray(reader BufferedReader) (bool, error) {
	// allow some leading whitespace
	bytes, err := reader.Peek(128)
	if err != nil {
		// if we have less than 128 bytes and we get an EOF, we will just look at the bytes
		// returned below
		if !errors.Is(err, io.EOF) {
			return false, fmt.Errorf("peek: %w", err)
		}
	}
	for _, b := range bytes {
		switch b {
		case '{', '[':
			return true, nil
		case ' ', '\t', '\n', '\r':
			// allow some leading whitespace
			continue
		default:
			return false, nil
		}
	}
	return false, nil
}

// Parse parses the JSON stream into a sequence of log records. The JSON stream is
// expected be either:
//
// 1. an array of log records
//
// 2. a single object with a "Records" key that contains an array of log records
//
// The parser will return an error if the stream is not valid. It will return
// ErrNotArrayOrKnownObject if the stream does not contain a valid array or object with a
// "Records" key.
func (p *jsonParser) Parse(_ context.Context, startOffset int64) (logs iter.Seq2[any, error], err error) {
	// Read the first object
	tok, err := p.decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("read first token: %w", err)
	}

	switch {
	case tok == json.Delim('['):
		// json structure is an array
		return p.yieldArray(startOffset), nil

	case tok == json.Delim('{'):
		// json structure is an object, find and yield the "Records" array containing log
		// records

		// iterate through key/value pairs
		for p.decoder.More() {
			// key
			tok, err := p.decoder.Token()
			if err != nil {
				return nil, fmt.Errorf("read token: %w", err)
			}
			key, ok := tok.(string)
			if !ok {
				// non-string key?
				continue
			}

			if key != "Records" {
				// we only look for Records in the first 4096 bytes
				if p.decoder.InputOffset() > maxRecordsSearchBytes {
					return nil, ErrNotArrayOrKnownObject
				}

				// skip the non-"Records" value
				if err := skipValue(p.decoder, maxRecordsSearchBytes); err != nil {
					return nil, fmt.Errorf("skip value: %w", err)
				}
				continue
			}

			// "Records" value
			tok, err = p.decoder.Token()
			if err != nil {
				return nil, fmt.Errorf("read token: %w", err)
			}
			switch tok {
			case json.Delim('['):
				return p.yieldArray(startOffset), nil

			default:
				// "Records" exists but is not an array
				return nil, ErrNotArrayOrKnownObject
			}
		}

		// we didn't find a top level array of log records or a "Records" key with an array of
		// log records
		return nil, ErrNotArrayOrKnownObject

	default:
		// not an array or object with a known key
		return nil, ErrNotArrayOrKnownObject
	}
}

func (p *jsonParser) Offset() int64 {
	return p.decoder.InputOffset()
}

func skipValue(decoder *json.Decoder, maxBytes int64) error {
	if decoder.InputOffset() > maxBytes {
		return ErrNotArrayOrKnownObject
	}

	// Read the next token to determine what we're skipping
	tok, err := decoder.Token()
	if err != nil {
		return err
	}

	switch delim := tok.(type) {
	case json.Delim:
		// If it's a delimiter, we need to skip everything inside
		switch delim {
		case '{', '[':
			// For each opening, keep skipping values until we find the matching closing
			for decoder.More() {
				if err := skipValue(decoder, maxBytes); err != nil {
					return err
				}
			}
			// Consume the closing delimiter
			_, err := decoder.Token()
			return err
		}
	}
	// If it's not a delimiter, it's a primitive value, so nothing more to skip
	return nil
}

func (p *jsonParser) yieldArray(startOffset int64) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		// Iterate through the array
		for p.decoder.More() {
			var record map[string]any
			currentOffset := p.decoder.InputOffset()

			if err := p.decoder.Decode(&record); err != nil {
				// normal end of file
				if errors.Is(err, io.EOF) {
					return
				}
				// unexpected end of file, not much we can do here
				if errors.Is(err, io.ErrUnexpectedEOF) {
					return
				}
				// unexpected error, return it
				if !yield(nil, fmt.Errorf("decode record: %w", err)) {
					return
				}
			} else {
				// if we haven't hit the start offset, skip the record
				if currentOffset < startOffset {
					continue
				}
				if !yield(record, nil) {
					return
				}
			}
		}
	}
}

// AppendLogBody appends the log record to the log record body using FromRaw.
func (p *jsonParser) AppendLogBody(_ context.Context, lr plog.LogRecord, record any) error {
	return lr.Body().FromRaw(record)
}
