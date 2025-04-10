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

// Package windows contains constants and functions for the Windows operating system.
package windows

import "syscall"

var (
	// ErrorInvalidHandle is returned when the handle is invalid
	ErrorInvalidHandle = syscall.Errno(6)
	// ErrorInvalidParameter is returned when the parameter is invalid or some other obscure windows error, look up the C++ error code for more details
	ErrorInvalidParameter = syscall.Errno(87)
	// ErrorInsufficientBuffer is returned when the buffer is insufficient from windows
	ErrorInsufficientBuffer = syscall.Errno(122)
	// ErrorNotFound is returned when the resource is not found
	ErrorNotFound = syscall.Errno(1168)
	// ErrorWMIInstanceNotFound is returned when the WMI instance is not found
	ErrorWMIInstanceNotFound = syscall.Errno(4021)
	// ErrorEVTInvalidEventData is returned when the event data is invalid
	ErrorEVTInvalidEventData = syscall.Errno(15005)
)
