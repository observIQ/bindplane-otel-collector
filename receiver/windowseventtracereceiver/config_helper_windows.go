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

package windowseventtracereceiver

import (
	"fmt"

	"github.com/observiq/bindplane-otel-contrib/receiver/windowseventtracereceiver/internal/etw/advapi32"
	"golang.org/x/sys/windows"
)

func validateProviderGUID(name string) error {
	_, err := windows.GUIDFromString(name)
	if err != nil {
		return fmt.Errorf("provider %q looks like a GUID but is not valid; expected format {xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx}", name)
	}
	return nil
}

func (l TraceLevelString) toTraceLevel() advapi32.TraceLevel {
	switch l {
	case LevelVerbose:
		return advapi32.TRACE_LEVEL_VERBOSE
	case LevelInformational:
		return advapi32.TRACE_LEVEL_INFORMATION
	case LevelWarning:
		return advapi32.TRACE_LEVEL_WARNING
	case LevelError:
		return advapi32.TRACE_LEVEL_ERROR
	case LevelCritical:
		return advapi32.TRACE_LEVEL_CRITICAL
	case LevelNone:
		return advapi32.TRACE_LEVEL_NONE
	default:
		return advapi32.TRACE_LEVEL_INFORMATION
	}
}
