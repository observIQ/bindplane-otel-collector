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
	"iter"
	"strings"

	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// LogParser is an interface that can parse a log stream into a sequence of log records
// and can also append a single log body to a LogRecord.
type LogParser interface {
	Parse(ctx context.Context) (logs iter.Seq2[any, error], err error)
	AppendLogBody(ctx context.Context, lr plog.LogRecord, record any) error
}

func newParser(ctx context.Context, stream logStream, reader *bufio.Reader) (parser LogParser, err error) {
	// if we're not trying to parse as JSON, use the line parser
	if !stream.tryJSON {
		return NewLineParser(reader), nil
	}

	isJSON, err := isJSON(ctx, stream, reader)
	if err != nil {
		// don't fail if the file is not json
		stream.logger.Warn("failed to check if is json", zap.Error(err))
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
		stream.logger.Warn("failed to check if starts with json object or array", zap.Error(err))
		return false, nil
	}

	return startsWithJSONObjectOrArray, nil
}

func isJSONExtension(name string) bool {
	return strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".json.gz")
}

func isJSONContentType(contentType *string) bool {
	return contentType != nil && strings.HasPrefix(*contentType, "application/json")
}
