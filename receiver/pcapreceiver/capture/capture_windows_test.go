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

//go:build windows

package capture

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildCaptureCommand_Windows_DefaultExe(t *testing.T) {
	cmd := BuildCaptureCommand("1", "tcp port 443", 65535, true)
	require.NotNil(t, cmd)
	require.Equal(t, []string{"-i", "1", "-q", "-F", "pcap", "-w", "-", "-s", "65535", "-f", "tcp port 443"}, cmd.Args[1:])
}

func TestBuildCaptureCommand_Windows_WithExecutablePath(t *testing.T) {
	exe := filepath.Join("C:", "Program Files", "Wireshark", "dumpcap.exe")
	cmd := BuildCaptureCommandWithExe(exe, "Ethernet 2", "udp port 53", 1024, false)
	require.NotNil(t, cmd)
	require.Equal(t, exe, cmd.Path)
	require.Equal(t, []string{"-i", "Ethernet 2", "-q", "-F", "pcap", "-w", "-", "-p", "-s", "1024", "-f", "udp port 53"}, cmd.Args[1:])
}
