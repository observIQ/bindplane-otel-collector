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
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"iter"
	"strings"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type logParser interface {
	Parse(ctx context.Context) (logs iter.Seq2[any, error], err error)
	AppendLogBody(ctx context.Context, lr plog.LogRecord, record any) error
}

type logStream struct {
	name            string
	contentEncoding *string
	contentType     *string
	body            io.ReadCloser
	maxLogSize      int
	logger          *zap.Logger
	loggerFields    []zap.Field
}

func newParser(ctx context.Context, stream logStream) (parser logParser, err error) {
	reader, err := streamReader(ctx, stream)
	if err != nil {
		return nil, fmt.Errorf("create stream reader: %w", err)
	}

	isJSON, err := isJSON(ctx, stream, reader)
	if err != nil {
		// don't fail if the file is not json
		fields := append([]zap.Field{zap.Error(err)}, stream.loggerFields...)
		stream.logger.Warn("failed to check if is json", fields...)
		isJSON = false
	}

	if isJSON {
		return NewJSONParser(reader), nil
	}
	return NewLineParser(reader), nil
}

func isJSON(_ context.Context, stream logStream, reader *bufio.Reader) (bool, error) {
	// check if the file extension or content type is json
	if !isJSONExtension(stream.name) && !isJSONContentType(stream.contentType) {
		return false, nil
	}

	// check if the stream starts with a json object or array
	startsWithJSONObjectOrArray, err := StartsWithJSONObjectOrArray(reader)
	if err != nil {
		return false, fmt.Errorf("check if starts with json object or array: %w", err)
	}

	return startsWithJSONObjectOrArray, nil
}

func isJSONExtension(name string) bool {
	return strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".json.gz")
}

func isJSONContentType(contentType *string) bool {
	return contentType != nil && strings.HasPrefix(*contentType, "application/json")
}

func streamReader(_ context.Context, stream logStream) (reader *bufio.Reader, err error) {
	// Check if content is gzipped and decompress if needed
	if stream.contentEncoding != nil {
		switch *stream.contentEncoding {
		case "gzip":
			gzipReader, err := gzip.NewReader(stream.body)
			if err != nil {
				return nil, fmt.Errorf("create gzip reader: %w", err)
			}
			return bufio.NewReaderSize(gzipReader, stream.maxLogSize), nil

		default:
			fields := append([]zap.Field{zap.String("content_encoding", *stream.contentEncoding)}, stream.loggerFields...)
			stream.logger.Warn("unsupported content encoding", fields...)
			return bufio.NewReaderSize(stream.body, stream.maxLogSize), nil
		}
	}

	if strings.HasSuffix(stream.name, ".gz") {
		gzipReader, err := gzip.NewReader(stream.body)
		if err != nil {
			return nil, fmt.Errorf("create gzip reader: %w", err)
		}
		return bufio.NewReaderSize(gzipReader, stream.maxLogSize), nil
	}

	return bufio.NewReaderSize(stream.body, stream.maxLogSize), nil
}
