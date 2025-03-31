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

// Package etw provides a pure Go implementation for accessing Windows Event Tracing (ETW).
//
// This package is designed to create ETW sessions, enable providers, and consume ETW events
// without requiring CGO. It uses the syscall package to interact with the Windows ETW APIs.
//
// The main components are:
// - Session: Manages ETW tracing sessions
// - Provider: Represents ETW providers that can be enabled in a session
// - Consumer: Consumes events from a session
package etw
