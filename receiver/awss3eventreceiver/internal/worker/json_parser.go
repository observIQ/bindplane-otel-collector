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
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"

	"go.opentelemetry.io/collector/pdata/plog"
)

type jsonParser struct {
	reader *bufio.Reader
}

var _ logParser = (*jsonParser)(nil)

// NewJSONParser creates a new JSON parser.
func NewJSONParser(reader *bufio.Reader) *jsonParser {
	return &jsonParser{
		reader: reader,
	}
}

// StartsWithJSONObjectOrArray returns true if the reader starts with a JSON object or
// array, allowing some space before the starting delimiter. It uses Peek and will not
// move the reader.
func StartsWithJSONObjectOrArray(reader *bufio.Reader) (bool, error) {
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
// 2. a single object where the value of the first key is an array of log
// records.
//
// The parser will return an error if the JSON stream is not valid.
func (p *jsonParser) Parse(_ context.Context) (logs iter.Seq2[any, error], err error) {
	decoder := json.NewDecoder(p.reader)

	// Read the first object
	tok, err := decoder.Token()
	if err != nil {
		return nil, fmt.Errorf("read first token: %w", err)
	}

	switch {
	case tok == json.Delim('['):
		// json structure is an array
		return p.yieldArray(decoder), nil

	case tok == json.Delim('{'):
		// json structure is an object, find and yield the first key containing an array of
		// log records

		// iterate through key/value pairs
		for decoder.More() {
			// key
			tok, err := decoder.Token()
			if err != nil {
				return nil, fmt.Errorf("read token: %w", err)
			}
			_, ok := tok.(string)
			if !ok {
				// non-string key?
				continue
			}

			// value
			tok, err = decoder.Token()
			if err != nil {
				return nil, fmt.Errorf("read token: %w", err)
			}
			switch tok {
			case json.Delim('['):
				return p.yieldArray(decoder), nil

			case json.Delim('{'):
				if err := skipObject(decoder); err != nil {
					return nil, fmt.Errorf("skip object: %w", err)
				}
			default:
				// skip non-array/object values
			}
		}
		return nil, fmt.Errorf("expected array of log records in one of the values")

	default:
		return nil, fmt.Errorf("expected array or object, got %s", tok)
	}
}

func skipObject(decoder *json.Decoder) error {
	for decoder.More() {
		// skip the key
		tok, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("read token: %w", err)
		}
		fmt.Printf("skipping tok: %s\n", tok)
		// skip the value
		if err := skipValue(decoder); err != nil {
			return err
		}
	}
	// consume the closing brace
	_, err := decoder.Token()
	return err
}

func skipValue(decoder *json.Decoder) error {
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
				if err := skipValue(decoder); err != nil {
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

func (p *jsonParser) yieldArray(decoder *json.Decoder) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		// Iterate through the array
		for decoder.More() {
			var record map[string]any

			if err := decoder.Decode(&record); err != nil {
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
