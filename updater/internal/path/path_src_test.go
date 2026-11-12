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

package path

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const lssrcCustom = `#subsysname:synonym:cmdargs:path:uid:auditid:standin:standout:standerr:action:multi:contact:svrkey:svrmtype:priority:signorm:sigforce:display:waittime:grpname:
observiq-otel-collector::--config /srv/bdot-custom/config.yaml:/srv/bdot-custom/dynatrace-bindplane-otel-collector:0:0:/dev/console:/dev/console:/dev/console:-R:-Q:-S:0:0:20:15:9:-d:20:oiqcollector:
`

func TestSrcInstallDir(t *testing.T) {
	const def = "/opt/observiq-otel-collector"
	tests := []struct {
		name     string
		home     string
		lssrcOut string
		want     string
	}{
		{name: "home from the SRC environment wins", home: "/srv/env-home", lssrcOut: lssrcCustom, want: "/srv/env-home"},
		{name: "home is cleaned", home: "/srv/env/", want: "/srv/env"},
		{name: "program directory from lssrc", lssrcOut: lssrcCustom, want: "/srv/bdot-custom"},
		{name: "subsystem not defined", lssrcOut: "", want: def},
		{name: "header only", lssrcOut: "#subsysname:synonym:cmdargs:path:\n", want: def},
		{name: "colon in a field", lssrcOut: "#subsysname:synonym:cmdargs:path:uid:\nx::--config /a:b/c.yaml:/srv/a:b/collector:0:\n", want: def},
		{name: "relative program path ignored", lssrcOut: "#subsysname:synonym:cmdargs:path:\nx::a:collector:0:\n", want: def},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, srcInstallDir(tc.home, []byte(tc.lssrcOut), def))
		})
	}
}
