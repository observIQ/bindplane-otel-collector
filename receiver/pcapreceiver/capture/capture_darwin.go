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

//go:build darwin

// Package capture provides functions to build capture commands for different platforms.
package capture

import (
	"fmt"
	"os/exec"
	"strings"
)

// BuildCaptureCommand builds the tcpdump command for macOS
func BuildCaptureCommand(iface, filter string, snaplen int, promisc bool) *exec.Cmd {
	args := []string{
		"-i", iface, // Interface
		"-n",  // Don't resolve hostnames
		"-xx", // Print packet data in hex with link-level headers
		"-l",  // Line buffered output
	}

	// Add --apple-md-print I flag to include interface name in output
	// Used to parse the interface name when capturing from "any" interface
	args = append(args, "--apple-md-print", "I")

	// Add -p flag to disable promiscuous mode if requested
	if !promisc {
		args = append(args, "-p")
	}

	// Add snapshot length
	args = append(args, "-s", fmt.Sprintf("%d", snaplen))

	// Add filter if specified
	if filter != "" {
		// Split filter into words for proper argument passing
		filterParts := strings.Fields(filter)
		args = append(args, filterParts...)
	}

	return exec.Command("tcpdump", args...) // #nosec G204 - interface and filter are validated by config
}
