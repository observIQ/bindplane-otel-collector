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
	"strings"

	"go.uber.org/zap"
)

type logStream struct {
	name            string
	contentEncoding *string
	contentType     *string
	body            io.ReadCloser
	maxLogSize      int
	logger          *zap.Logger
	tryJSON         bool
}

func (stream *logStream) BufferedReader(_ context.Context) (reader *bufio.Reader, err error) {
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
			stream.logger.Warn("unsupported content encoding", zap.String("content_encoding", *stream.contentEncoding))
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
