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

// platformDefaultSnaplen returns the default snap length for Windows
func platformDefaultSnaplen() int32 {
	return 262144 // 256KB for Windows/Npcap
}

// checkPlatformPermissions checks for required permissions on Windows
func checkPlatformPermissions() error {
	// To be implemented in Phase 5
	// Check for Npcap installation and admin privileges
	return nil
}

