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

package utils

import (
	"fmt"
	"strings"
)

// Take returns the first n bytes from data, along with the remainder (remainder is the first returned value).
// It mimics nom::take(n) from rust
func Take(data []byte, n int) ([]byte, []byte, error) {
	if len(data) < n {
		return nil, nil, fmt.Errorf("not enough bytes")
	}
	return data[n:], data[:n], nil
}

// ExtractStringSize extracts a string from the data based on the message size
// Returns the remaining data, the string, and an error if the string is not found or if the message size is too large
func ExtractStringSize(data []byte, messageSize uint64) ([]byte, string, error) {
	const nullString = uint64(0)
	if messageSize == nullString {
		return data, "(null)", nil
	}

	// Convert u64 to int, checking for overflow
	if messageSize > uint64(^uint(0)>>1) { // Check if larger than max int
		return data, "", fmt.Errorf("failed to extract string size: u64 is bigger than system usize")
	}

	size := int(messageSize)
	if len(data) < size {
		// Not enough data, take what we have
		input, path, _ := Take(data, len(data))
		pathString := string(path)
		return input, strings.TrimRight(pathString, "\x00"), nil
	}

	// Extract the string bytes
	input, path, _ := Take(data, size)
	pathString := string(path)
	// Remove trailing null bytes
	pathString = strings.TrimRight(pathString, "\x00")

	return input, pathString, nil
}

// paddingSizeFour calculates 4-byte alignment padding (equivalent to Rust's padding_size_four)
func PaddingSizeFour(dataSize uint64) uint64 {
	return PaddingSize(dataSize, 4)
}

// paddingSize calculates padding based on alignment (equivalent to Rust's padding_size)
func PaddingSize(dataSize uint64, alignment uint64) uint64 {
	return (alignment - (dataSize & (alignment - 1))) & (alignment - 1)
}
