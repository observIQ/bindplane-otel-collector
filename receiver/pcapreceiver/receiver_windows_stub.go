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

//go:build !windows

package pcapreceiver

import (
	"context"
	"io"
)

// readPacketsWindows is a stub for non-Windows builds
// This should never be called since readPackets checks runtime.GOOS first
func (r *pcapReceiver) readPacketsWindows(_ context.Context, _ io.ReadCloser) {
	// This should never be called on non-Windows platforms
	// The runtime check in readPackets should prevent this
	panic("readPacketsWindows called on non-Windows platform - this is a programming error")
}
