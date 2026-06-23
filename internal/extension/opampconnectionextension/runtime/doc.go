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

// Package runtime is the managed-agent entry point invoked by the
// collector binary's main.go. Run(Options) parses no flags of its own;
// callers pass the resolved collector/manager/logging paths and ocb
// factory set, and Run owns the rest: logger setup, feature-gate
// plumbing, manager.yaml bootstrap from env vars, rollback handling,
// and the managed-vs-standalone dispatch.
package runtime
