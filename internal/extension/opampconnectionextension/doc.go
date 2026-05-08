// Copyright  observIQ, Inc.
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

// Package opampconnectionextension provides an extension that exposes the
// OpAMP connection established by the opamp package to other collector
// components. It implements the opampcustommessages.CustomCapabilityRegistry
// interface so that processors and other extensions can register custom
// capabilities, send custom messages, and receive custom messages from the
// OpAMP server using the existing connection managed outside of the
// collector.
package opampconnectionextension
