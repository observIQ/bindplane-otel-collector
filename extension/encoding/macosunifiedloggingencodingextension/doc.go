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

// Package macosunifiedloggingencodingextension provides an encoding extension
// for decoding macOS Unified Logging binary files (tracev3 format).
//
// This extension reads binary data from macOS Unified Logging archives and
// converts them into OpenTelemetry log records. It is designed to work with
// the macOS Unified Logging receiver to parse system logs on macOS platforms.
//
// The extension implements the LogsUnmarshalerExtension interface to decode
// binary log data into structured log records that can be processed by the
// OpenTelemetry Collector.
package macosunifiedloggingencodingextension // import "github.com/observiq/bindplane-otel-collector/extension/encoding/macosunifiedloggingencodingextension"
