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
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"strings"

	"go.uber.org/zap"
)

// LogStream is a struct containing the information about a stream of logs.
type LogStream struct {
	Name            string
	ContentEncoding *string
	ContentType     *string
	Body            io.ReadCloser
	MaxLogSize      int
	Logger          *zap.Logger
	TryDecoding     bool
}

// BufferedReader returns a BufferedReader for the log stream. If the content is gzipped
// (content-encoding: gzip or ends with .gz), it will be decompressed.
func (stream *LogStream) BufferedReader(_ context.Context) (BufferedReader, error) {
	// Check if content is gzipped and decompress if needed
	if stream.ContentEncoding != nil {
		switch *stream.ContentEncoding {
		case "gzip":
			gzipReader, err := gzip.NewReader(stream.Body)
			if err != nil {
				return nil, fmt.Errorf("create gzip reader: %w", err)
			}
			return NewBufferedReader(gzipReader, stream.MaxLogSize), nil

		default:
			stream.Logger.Warn("unsupported content encoding", zap.String("content_encoding", *stream.ContentEncoding))
			return NewBufferedReader(stream.Body, stream.MaxLogSize), nil
		}
	}

	if strings.HasSuffix(stream.Name, ".gz") {
		gzipReader, err := gzip.NewReader(stream.Body)
		if err != nil {
			return nil, fmt.Errorf("create gzip reader: %w", err)
		}
		return NewBufferedReader(gzipReader, stream.MaxLogSize), nil
	}

	return NewBufferedReader(stream.Body, stream.MaxLogSize), nil
}
