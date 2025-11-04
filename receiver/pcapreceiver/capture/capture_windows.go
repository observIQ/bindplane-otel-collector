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
	"fmt"
	"os/exec"
	"path/filepath"
)

// BuildCaptureCommand builds the dumpcap command for Windows (Wireshark)
// iface may be a numeric index (from "dumpcap -D") or an interface name.
// If executablePath is non-empty, it will be used as the dumpcap executable;
// otherwise "dumpcap.exe" is used (typically from Wireshark installation).
func BuildCaptureCommandWithExe(executablePath, iface, filter string, snaplen int, promisc bool) *exec.Cmd {
	args := []string{
		"-i", iface, // Interface index or name
		"-q",         // Don't report packet counts
		"-F", "pcap", // Output in pcap format
		"-w", "-", // Write to stdout
	}

	if !promisc {
		args = append(args, "-p")
	}

	args = append(args, "-s", fmt.Sprintf("%d", snaplen))

	if filter != "" {
		// dumpcap uses -f flag with BPF filter syntax (same as tcpdump)
		// Pass filter as a single string argument (not split)
		args = append(args, "-f", filter)
	}

	exe := executablePath
	if exe == "" {
		// Try common Wireshark installation paths
		commonPaths := []string{
			`C:\Program Files\Wireshark\dumpcap.exe`,
			`C:\Program Files (x86)\Wireshark\dumpcap.exe`,
		}
		for _, path := range commonPaths {
			if _, err := exec.LookPath(path); err == nil {
				exe = path
				break
			}
		}
		if exe == "" {
			exe = "dumpcap"
		}
	} else {
		exe = filepath.Clean(executablePath)
	}

	return exec.Command(exe, args...) // #nosec G204 - args validated by config
}

// BuildCaptureCommand builds the dumpcap command using default executable lookup
func BuildCaptureCommand(iface, filter string, snaplen int, promisc bool) *exec.Cmd {
	return BuildCaptureCommandWithExe("", iface, filter, snaplen, promisc)
}
