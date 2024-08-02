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
//go:build tools

// Package tools exists to provide imports for tools used in development
package tools

import (
	_ "github.com/client9/misspell/cmd/misspell"
	_ "github.com/google/addlicense"
	_ "github.com/mgechev/revive"
	_ "github.com/open-telemetry/opentelemetry-collector-contrib/cmd/opampsupervisor"
	_ "github.com/securego/gosec/v2/cmd/gosec"
	_ "github.com/uw-labs/lichen"
	_ "github.com/vektra/mockery/v2"
	_ "go.opentelemetry.io/collector/cmd/builder"
	_ "go.opentelemetry.io/collector/cmd/mdatagen"
	_ "golang.org/x/tools/cmd/goimports"
)
