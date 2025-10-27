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
	// Interface specifies the network interface to capture from
	// If empty, auto-detects the first available interface
	Interface string `mapstructure:"interface"`

	// SnapLength is the maximum bytes to capture per packet
	// Default: 65535 (full packet)
	SnapLength int32 `mapstructure:"snap_length"`

	// BPFFilter is a Berkeley Packet Filter expression to filter packets
	// Example: "tcp port 80"
	BPFFilter string `mapstructure:"bpf_filter"`

	// Promiscuous enables promiscuous mode on the interface
	// Default: true
	Promiscuous bool `mapstructure:"promiscuous"`
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// To be implemented in Phase 4
	return nil
}

