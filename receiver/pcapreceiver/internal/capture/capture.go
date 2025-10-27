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

package capture

import (
	"context"

	"github.com/google/gopacket"
)

// Capturer defines the interface for packet capture
type Capturer interface {
	// Start begins packet capture
	Start(ctx context.Context) error
	
	// Packets returns a channel of captured packets
	Packets() <-chan gopacket.Packet
	
	// Stats returns capture statistics
	Stats() (*Stats, error)
	
	// Close stops capture and releases resources
	Close() error
}

// Stats contains packet capture statistics
type Stats struct {
	PacketsReceived uint64
	PacketsDropped  uint64
	InterfaceName   string
}

// Config contains configuration for the capturer
type Config struct {
	Interface   string
	SnapLength  int32
	BPFFilter   string
	Promiscuous bool
}

// NewCapturer creates a new packet capturer
func NewCapturer(config Config) (Capturer, error) {
	// To be implemented in Phase 5
	return nil, nil
}

