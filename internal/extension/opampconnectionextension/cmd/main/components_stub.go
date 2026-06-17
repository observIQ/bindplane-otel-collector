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

// components() is normally provided by ocb's generated `components.go` in
// ./build/. This stub lets the file compile in its source location so the
// extension module's own `go test ./...` / lint walk doesn't break. The
// Makefile `agent` target copies main.go (not this file) into the
// ocb-generated build dir, where the generated definition takes over.
package main

import (
	"errors"

	"go.opentelemetry.io/collector/otelcol"
)

func components() (otelcol.Factories, error) {
	return otelcol.Factories{}, errors.New("components(): source-tree stub — build via `make agent` to get the ocb-generated factory set")
}
