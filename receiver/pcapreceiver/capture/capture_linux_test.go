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

//go:build linux

package capture

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildCaptureCommand_Linux(t *testing.T) {
	tests := []struct {
		name     string
		iface    string
		filter   string
		snaplen  int
		promisc  bool
		wantArgs []string
	}{
		{
			name:     "basic command with interface",
			iface:    "eth0",
			filter:   "",
			snaplen:  65535,
			promisc:  true,
			wantArgs: []string{"-i", "eth0", "-n", "-xx", "-l", "-s", "65535"},
		},
		{
			name:     "command with filter",
			iface:    "eth0",
			filter:   "tcp port 443",
			snaplen:  65535,
			promisc:  true,
			wantArgs: []string{"-i", "eth0", "-n", "-xx", "-l", "-s", "65535", "tcp", "port", "443"},
		},
		{
			name:     "command with complex filter",
			iface:    "ens160",
			filter:   "tcp port 80 or udp port 53",
			snaplen:  1500,
			promisc:  true,
			wantArgs: []string{"-i", "ens160", "-n", "-xx", "-l", "-s", "1500", "tcp", "port", "80", "or", "udp", "port", "53"},
		},
		{
			name:     "command without promiscuous mode",
			iface:    "eth0",
			filter:   "",
			snaplen:  65535,
			promisc:  false,
			wantArgs: []string{"-i", "eth0", "-n", "-xx", "-l", "-p", "-s", "65535"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := BuildCaptureCommand(tt.iface, tt.filter, tt.snaplen, tt.promisc)
			require.NotNil(t, cmd)
			require.True(t, strings.HasSuffix(cmd.Path, "tcpdump"))
			require.Equal(t, tt.wantArgs, cmd.Args[1:])
		})
	}
}
