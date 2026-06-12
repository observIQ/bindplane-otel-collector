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

// Package opamp contains configurations and protocol implementations to handle OpAmp communication.
package opamp

// generate directives for interfaces in third-party packages:
//go:generate mockery --srcpkg github.com/open-telemetry/opamp-go/client/types --name PackagesStateProvider --filename mock_packages_state_provider.go --structname MockPackagesStateProvider
//go:generate mockery --srcpkg github.com/open-telemetry/opamp-go/client --name OpAMPClient --filename mock_opamp_client.go --structname MockOpAMPClient
