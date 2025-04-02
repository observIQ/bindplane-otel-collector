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

//go:build windows

package etw

import (
	"fmt"
	"syscall"

	"github.com/observiq/bindplane-otel-collector/receiver/windowseventtracereceiver/internal/etw/advapi32"
	"golang.org/x/sys/windows"
)

// Define our own version of the EnableTrace function that works with simple parameters
// This avoids the need to create the exact same struct as in the advapi32 package
func enableProvider(handle syscall.Handle, providerGUID *windows.GUID) error {

	// Define the constants we need
	const (
		EVENT_CONTROL_CODE_ENABLE_PROVIDER = 1
		TRACE_LEVEL_INFORMATION            = 4
	)

	params := advapi32.EnableTraceParameters{Version: 2}
	r1, err := advapi32.EnableTrace(
		handle,
		providerGUID,
		EVENT_CONTROL_CODE_ENABLE_PROVIDER,
		TRACE_LEVEL_INFORMATION,
		0,
		0,
		0,
		&params,
	)

	if r1 == 87 {
		// retry
		r1, err = advapi32.EnableTrace(
			handle,
			providerGUID,
			EVENT_CONTROL_CODE_ENABLE_PROVIDER,
			TRACE_LEVEL_INFORMATION,
			0,
			0,
			0,
			&params,
		)
		if r1 != 0 {
			return fmt.Errorf("EnableTrace failed(%d): %w", r1, err)
		}
	}

	if r1 != 0 {
		return fmt.Errorf("EnableTrace failed(%d): %w", r1, err)
	}

	return nil
}
