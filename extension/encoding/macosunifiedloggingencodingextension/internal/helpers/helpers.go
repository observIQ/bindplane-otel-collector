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

// Package helpers provides helper functions for parsing Unified Log data
package helpers // import "github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension/internal/helpers"

import (
	"fmt"
	"strings"
)

// Take returns the first n bytes from data, along with the remainder (remainder is the first returned value).
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

// ExtractString extracts a string from the data
func ExtractString(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("cannot extract string: Empty input")
	}

	// If message data does not end with null terminator (0)
	// just grab everything and convert what we have to string
	lastByte := data[len(data)-1]
	if lastByte != 0 {
		// Convert entire data to string
		return string(data), nil
	}

	// Find the null terminator
	for i, b := range data {
		if b == 0 {
			// Return string up to (but not including) the null terminator
			return string(data[:i]), nil
		}
	}

	// Fallback (shouldn't reach here given the check above)
	return "", fmt.Errorf("could not extract string %s", data)
}

// PaddingSizeFour calculates 4-byte alignment padding (equivalent to Rust's padding_size_four)
func PaddingSizeFour(dataSize uint64) uint64 {
	return PaddingSize(dataSize, 4)
}

// PaddingSize calculates padding based on alignment (equivalent to Rust's padding_size)
func PaddingSize(dataSize uint64, alignment uint64) uint64 {
	return (alignment - (dataSize & (alignment - 1))) & (alignment - 1)
}

// AnticipatedPaddingSize returns the padding bytes required to align
// a contiguous block of data of size dataCount*dataSize to the given alignment.
// It mirrors PaddingSize but for a computed total.
func AnticipatedPaddingSize(dataCount uint64, dataSize uint64, alignment uint64) uint64 {
	totalSize := dataCount * dataSize
	return PaddingSize(totalSize, alignment)
}
