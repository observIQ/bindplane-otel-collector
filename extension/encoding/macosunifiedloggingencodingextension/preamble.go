// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package macosunifiedloggingencodingextension

import (
	"encoding/binary"
	"fmt"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/helpers"
)

type LogPreamble struct {
	ChunkTag      uint32
	ChunkSubtag   uint32
	ChunkDataSize uint64
}

// DetectPreamble gets the 1st 16 bytes of the data and parses it into a LogPreamble
// Doesn't consume the input
func DetectPreamble(data []byte) (LogPreamble, error) {
	preamble, _, err := ParsePreamble(data)
	return preamble, err
}

// ParsePreamble parses the 1st 16 bytes of the data into a LogPreamble
// Consumes the input
func ParsePreamble(data []byte) (LogPreamble, []byte, error) {
	var chunkTag, chunkSubTag, chunkDataSize []byte
	var err error

	data, chunkTag, err = helpers.Take(data, 4)
	if err != nil {
		return LogPreamble{}, data, fmt.Errorf("failed to parse preamble chunk tag: %w", err)
	}
	data, chunkSubTag, err = helpers.Take(data, 4)
	if err != nil {
		return LogPreamble{}, data, fmt.Errorf("failed to parse preamble chunk subtag: %w", err)
	}
	data, chunkDataSize, err = helpers.Take(data, 8)
	if err != nil {
		return LogPreamble{}, data, fmt.Errorf("failed to parse preamble chunk data size: %w", err)
	}

	return LogPreamble{
		ChunkTag:      binary.LittleEndian.Uint32(chunkTag),
		ChunkSubtag:   binary.LittleEndian.Uint32(chunkSubTag),
		ChunkDataSize: binary.LittleEndian.Uint64(chunkDataSize),
	}, data, nil
}
