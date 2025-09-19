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
	"fmt"
)

// ParseStatedumpChunk parses a Statedump chunk (0x6003) containing system state information
func ParseStatedumpChunk(data []byte, entry *TraceV3Entry) {
	if len(data) < 16 {
		entry.Message = fmt.Sprintf("Statedump chunk too small: %d bytes", len(data))
		return
	}

	entry.Message = fmt.Sprintf("Statedump entry: system state data (%d bytes)", len(data))
	entry.Level = "Debug"
}
