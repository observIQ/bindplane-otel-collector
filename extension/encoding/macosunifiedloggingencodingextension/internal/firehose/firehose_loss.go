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

package firehose

import (
	"encoding/binary"
	"fmt"

	"github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/utils"
)

// Loss represents a parsed firehose loss entry
type Loss struct {
	// TODO: consider using time.Time instead of uint64 once timesync is re-examined
	StartTime uint64
	EndTime   uint64
	Count     uint64
}

// ParseFirehoseLoss parses a firehose loss entry
func ParseFirehoseLoss(data []byte) (Loss, []byte, error) {
	firehoseLoss := Loss{}

	data, startTime, err := utils.Take(data, 8)
	if err != nil {
		return firehoseLoss, data, fmt.Errorf("failed to read start time: %w", err)
	}
	data, endTime, err := utils.Take(data, 8)
	if err != nil {
		return firehoseLoss, data, fmt.Errorf("failed to read end time: %w", err)
	}
	data, count, err := utils.Take(data, 8)
	if err != nil {
		return firehoseLoss, data, fmt.Errorf("failed to read count: %w", err)
	}

	firehoseLoss.StartTime = binary.LittleEndian.Uint64(startTime)
	firehoseLoss.EndTime = binary.LittleEndian.Uint64(endTime)
	firehoseLoss.Count = binary.LittleEndian.Uint64(count)

	return firehoseLoss, data, nil
}
