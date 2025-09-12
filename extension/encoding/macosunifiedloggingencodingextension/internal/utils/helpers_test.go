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

package utils

import (
	"testing"
)

func TestExtractStringSize(t *testing.T) {
	data := []byte{55, 57, 54, 46, 49, 48, 48, 0}
	size := uint64(8)
	_, result, err := ExtractStringSize(data, size)
	if err != nil {
		t.Fatalf("failed to extract string size: %v", err)
	}
	if result != "796.100" {
		t.Fatalf("expected '796.100', got '%s'", result)
	}
}
