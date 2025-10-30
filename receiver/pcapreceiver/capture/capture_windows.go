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
	"strings"
)

// BuildCaptureCommand builds the windump command for Windows (Npcap)
// iface may be a numeric index (from "windump -D") or an interface name.
// If executablePath is non-empty, it will be used as the windump executable; otherwise "windump.exe" is used.
func BuildCaptureCommandWithExe(executablePath, iface, filter string, snaplen int, promisc bool) *exec.Cmd {
	args := []string{
		"-i", iface, // Interface index or name
		"-n",        // No name resolution
		"-xx",       // Hex dump including link-level
		"-l",        // Line buffered
	}

	if !promisc {
		args = append(args, "-p")
	}

	args = append(args, "-s", fmt.Sprintf("%d", snaplen))

	if filter != "" {
		args = append(args, strings.Fields(filter)...)
	}

	exe := executablePath
	if exe == "" {
		exe = "windump.exe"
	} else {
		exe = filepath.Clean(executablePath)
	}

	return exec.Command(exe, args...) // #nosec G204 - args validated by config
}

// BuildCaptureCommand builds the windump command using default executable lookup
func BuildCaptureCommandWindows(iface, filter string, snaplen int, promisc bool) *exec.Cmd {
	return BuildCaptureCommandWithExe("", iface, filter, snaplen, promisc)
}

