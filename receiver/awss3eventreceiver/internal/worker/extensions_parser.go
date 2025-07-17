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
	"io"
	"iter"

	"github.com/aws/aws-lambda-go/events"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
)

type extensionsParser struct {
	reader    BufferedReader
	extension component.Component
}

var _ LogParser = (*extensionsParser)(nil)

func NewExtensionsParser(reader BufferedReader, extension component.Component) LogParser {
	return &extensionsParser{reader: reader, extension: extension}
}

// Parse implements LogParser.
func (ep *extensionsParser) Parse(record events.S3EventRecord, maxLogsEmitted int, startOffset int64) (logs iter.Seq2[plog.Logs, error], err error) {
	var unmarshaler plog.Unmarshaler
	unmarshaler, _ = ep.extension.(plog.Unmarshaler)

	// read the entire body of the log record into a single byte array, extensions don't support streaming
	body := []byte{}
	for {
		lineBytes, _, err := ep.reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		body = append(body, lineBytes...)
	}

	unmarshalledLogs, err := unmarshaler.UnmarshalLogs(body)
	if err != nil {
		return nil, err
	}

	return func(yield func(plog.Logs, error) bool) {
		if !yield(unmarshalledLogs, nil) {
			return
		}
	}, nil
}

// Offset returns the current offset of the log stream.
func (ep *extensionsParser) Offset() int64 {
	panic("unimplemented")
}
