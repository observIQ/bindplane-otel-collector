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

// Package capture provides functions to build capture commands for different platforms.
package capture

import (
	"fmt"
	"os/exec"
	"strings"
)

// BuildCaptureCommand builds the tcpdump command for Linux
func BuildCaptureCommand(iface, filter string, snaplen int, promisc bool) *exec.Cmd {
	args := []string{
		"-i", iface, // Interface
		"-n",  // Don't resolve hostnames
		"-xx", // Print packet data in hex with link-level headers
		"-l",  // Line buffered output
	}

	// Add -p flag to disable promiscuous mode if requested
	if !promisc {
		args = append(args, "-p")
	}

	// Add snapshot length
	args = append(args, "-s", fmt.Sprintf("%d", snaplen))

	// Add filter if specified
	if filter != "" {
		filterParts := strings.Fields(filter)
		args = append(args, filterParts...)
	}

	return exec.Command("tcpdump", args...) // #nosec G204 - validated by config
}
