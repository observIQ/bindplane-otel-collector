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

import (
	"errors"
	"fmt"

	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

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
	var errs error

	// Validate snap length
	if c.SnapLength <= 0 {
		errs = errors.Join(errs, fmt.Errorf("snap_length must be greater than 0, got %d", c.SnapLength))
	}

	// Validate BPF filter syntax if provided
	if c.BPFFilter != "" {
		// Use pcap.CompileBPFFilter to validate syntax
		// We need to provide a link type - use Ethernet as default
		_, err := pcap.CompileBPFFilter(layers.LinkTypeEthernet, int(c.SnapLength), c.BPFFilter)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("invalid bpf_filter: %w", err))
		}
	}

	// Note: We don't validate interface existence here because:
	// 1. Empty interface means auto-detect
	// 2. Interfaces may not exist during config validation (e.g., in CI/tests)
	// 3. Interface validation will happen at Start() time with proper error reporting

	return errs
}

