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

package macosunifiedloggingencodingextension

import "encoding/binary"

type FirehoseLoss struct {
	// TODO: consider using time.Time instead of uint64 once timesync is re-examined
	StartTime uint64
	EndTime   uint64
	Count     uint64
}

func ParseFirehoseLoss(data []byte) (FirehoseLoss, []byte) {
	firehoseLoss := FirehoseLoss{}

	data, startTime, _ := Take(data, 8)
	data, endTime, _ := Take(data, 8)
	data, count, _ := Take(data, 8)

	firehoseLoss.StartTime = binary.LittleEndian.Uint64(startTime)
	firehoseLoss.EndTime = binary.LittleEndian.Uint64(endTime)
	firehoseLoss.Count = binary.LittleEndian.Uint64(count)

	return firehoseLoss, data
}
