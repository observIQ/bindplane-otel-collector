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

//go:build !windows

package capture

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildCaptureCommand_Unix(t *testing.T) {
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
			iface:    "en0",
			filter:   "",
			snaplen:  65535,
			promisc:  true,
			wantArgs: []string{"-i", "en0", "-n", "-xx", "-l", "-s", "65535"},
		},
		{
			name:     "command with filter",
			iface:    "en0",
			filter:   "tcp port 443",
			snaplen:  65535,
			promisc:  true,
			wantArgs: []string{"-i", "en0", "-n", "-xx", "-l", "-s", "65535", "tcp", "port", "443"},
		},
		{
			name:     "command with complex filter",
			iface:    "eth0",
			filter:   "tcp port 80 or udp port 53",
			snaplen:  1500,
			promisc:  true,
			wantArgs: []string{"-i", "eth0", "-n", "-xx", "-l", "-s", "1500", "tcp", "port", "80", "or", "udp", "port", "53"},
		},
		{
			name:     "command without promiscuous mode",
			iface:    "en0",
			filter:   "",
			snaplen:  65535,
			promisc:  false,
			wantArgs: []string{"-i", "en0", "-n", "-xx", "-l", "-p", "-s", "65535"},
		},
		{
			name:     "command with custom snaplen",
			iface:    "en0",
			filter:   "",
			snaplen:  1024,
			promisc:  true,
			wantArgs: []string{"-i", "en0", "-n", "-xx", "-l", "-s", "1024"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := BuildCaptureCommand(tt.iface, tt.filter, tt.snaplen, tt.promisc)
			require.NotNil(t, cmd)

			// Verify command is tcpdump (path may be absolute)
			require.True(t, strings.HasSuffix(cmd.Path, "tcpdump"), "Expected path to end with 'tcpdump', got: %s", cmd.Path)

			// Verify arguments
			require.Equal(t, tt.wantArgs, cmd.Args[1:]) // Skip cmd.Args[0] which is the program name
		})
	}
}

func TestBuildCaptureCommand_Unix_FilterWithParentheses(t *testing.T) {
	cmd := BuildCaptureCommand("en0", "(tcp port 80 or tcp port 443) and not src 192.168.1.1", 65535, true)
	require.NotNil(t, cmd)

	// Verify the filter arguments are properly split
	args := cmd.Args[1:]
	require.Contains(t, args, "(tcp")
	require.Contains(t, args, "and")
	require.Contains(t, args, "not")
}
