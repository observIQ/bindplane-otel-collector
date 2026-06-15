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
	"iter"
	"strings"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// LogParser is an interface that can parse a log stream into a sequence of log records
// and can also append a single log body to a LogRecord.
type LogParser interface {
	// Parse parses the log stream into a sequence of log records. The parser should return
	// an error if the stream is not valid.
	Parse(ctx context.Context, startOffset int64) (logs iter.Seq2[any, error], err error)

	// AppendLogBody appends a single log body to a LogRecord. Different parsers may result
	// in different log bodies so this is the responsibility of the parser.
	AppendLogBody(ctx context.Context, lr plog.LogRecord, record any) error

	// Offset returns the current offset of the log stream.
	Offset() int64
}

func newParser(ctx context.Context, stream LogStream, reader BufferedReader) (parser LogParser, err error) {
	// if we're not trying to decode, use the line parser
	if !stream.TryDecoding {
		return NewLineParser(reader), nil
	}

	// check for avro first
	isAvro, err := isAvroOcf(ctx, stream, reader)
	if err != nil {
		stream.Logger.Warn("failed to check if is avro", zap.Error(err))
		isAvro = false
	}
	if isAvro {
		return NewAvroOcfParser(reader, stream.Logger), nil
	}

	// check for json
	isJSON, err := isJSON(ctx, stream, reader)
	if err != nil {
		// don't fail if the file is not json
		stream.Logger.Warn("failed to check if is json", zap.Error(err))
		isJSON = false
	}

	if isJSON {
		return NewJSONParser(reader), nil
	}
	return NewLineParser(reader), nil
}

func isJSON(_ context.Context, stream LogStream, reader BufferedReader) (bool, error) {
	// check if the file extension or content type is json
	if !isJSONExtension(stream.Name) && !isJSONContentType(stream.ContentType) {
		return false, nil
	}

	// check if the stream starts with a json object or array
	startsWithJSONObjectOrArray, err := StartsWithJSONObjectOrArray(reader)
	if err != nil {
		stream.Logger.Warn("failed to check if starts with json object or array", zap.Error(err))
		return false, nil
	}

	return startsWithJSONObjectOrArray, nil
}

func isAvroOcf(_ context.Context, stream LogStream, reader BufferedReader) (bool, error) {
	// check if the file extension or content type is avro
	if !isAvroExtension(stream.Name) && !isAvroContentType(stream.ContentType) {
		return false, nil
	}

	// check if the stream starts with the avro ocf magic string
	startsWithAvroOcfMagic, err := StartsWithAvroOcfMagic(reader)
	if err != nil {
		stream.Logger.Warn("failed to check if starts with avro ocf magic", zap.Error(err))
		return false, nil
	}

	return startsWithAvroOcfMagic, nil
}

func isJSONExtension(name string) bool {
	return strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".json.gz")
}

func isJSONContentType(contentType *string) bool {
	return contentType != nil && strings.HasPrefix(*contentType, "application/json")
}

func isAvroExtension(name string) bool {
	return strings.HasSuffix(name, ".avro") || strings.HasSuffix(name, ".avro.gz")
}

func isAvroContentType(contentType *string) bool {
	return contentType != nil && strings.HasPrefix(*contentType, "application/avro")
}
