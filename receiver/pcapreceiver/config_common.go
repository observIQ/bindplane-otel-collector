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

package pcapreceiver

// Config defines configuration for the PCAP receiver
type Config struct {
	Interface      string `mapstructure:"interface"`       // Network interface to capture on (e.g., "en0")
	Filter         string `mapstructure:"filter"`          // BPF filter expression (optional)
	SnapLen        int    `mapstructure:"snaplen"`         // Snapshot length in bytes (default: 65535)
	Promiscuous    bool   `mapstructure:"promiscuous"`     // Enable promiscuous mode (default: true)
	ExecutablePath string `mapstructure:"executable_path"` // Windows only: optional override path to dumpcap.exe
	AddAttributes  bool   `mapstructure:"add_attributes"`  // Add network attributes to log records
}
