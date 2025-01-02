// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package ccid exposes a hardcoded UUID that is used to identify bindplane agents in Chronicle.
package ccid

import (
	"github.com/google/uuid"
)

// ChronicleCollectorID is a specific collector ID for Chronicle. It's used to identify bindplane agents in Chronicle.
var ChronicleCollectorID = uuid.MustParse("aaaa1111-aaaa-1111-aaaa-1111aaaa1111")
