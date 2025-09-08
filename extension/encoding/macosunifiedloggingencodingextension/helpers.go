// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package macosunifiedloggingencodingextension

import (
	"fmt"
	"strings"
)

// take returns the first n bytes from data, along with the remainder.
// It mimics nom::take(n) from rust
func Take(data []byte, n int) ([]byte, []byte, error) {
	if len(data) < n {
		return nil, nil, fmt.Errorf("not enough bytes")
	}
	return data[n:], data[:n], nil
}

func ExtractStringSize(data []byte, messageSize uint64) ([]byte, string, error) {
	const NULL_STRING = uint64(0)
	if messageSize == NULL_STRING {
		return data, "(null)", nil
	}

	// Convert u64 to int, checking for overflow
	if messageSize > uint64(^uint(0)>>1) { // Check if larger than max int
		return data, "", fmt.Errorf("failed to extract string size: u64 is bigger than system usize")
	}

	size := int(messageSize)
	if len(data) < size {
		// Not enough data, take what we have
		path, input, _ := Take(data, len(data))
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
func paddingSizeFour(dataSize uint64) uint64 {
	return paddingSize(dataSize, 4)
}

// paddingSize calculates padding based on alignment (equivalent to Rust's padding_size)
func paddingSize(dataSize uint64, alignment uint64) uint64 {
	return (alignment - (dataSize & (alignment - 1))) & (alignment - 1)
}
