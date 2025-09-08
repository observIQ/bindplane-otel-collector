// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package macosunifiedloggingencodingextension

import "fmt"

// take returns the first n bytes from data, along with the remainder.
// It mimics nom::take(n) from rust
func Take(data []byte, n int) ([]byte, []byte, error) {
	if len(data) < n {
		return nil, nil, fmt.Errorf("not enough bytes")
	}
	return data[n:], data[:n], nil
}
