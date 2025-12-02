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

package pcapreceiver

import (
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// PcapInterface defines the interface for pcap operations, allowing for mocking in tests
type PcapInterface interface {
	// FindAllDevs returns all available network devices
	FindAllDevs() ([]pcap.Interface, error)
	// OpenLive opens a device for live capture
	OpenLive(device string, snaplen int32, promisc bool, timeout time.Duration) (PcapHandle, error)
}

// PcapHandle defines the interface for a pcap capture handle
type PcapHandle interface {
	// SetBPFFilter sets a BPF filter on the handle
	SetBPFFilter(filter string) error
	// Close closes the handle
	Close()
	// LinkType returns the link type of the handle
	LinkType() layers.LinkType
	// ReadPacketData reads the next packet from the handle
	ReadPacketData() ([]byte, gopacket.CaptureInfo, error)
}

// realPcapInterface is the real implementation that calls the actual pcap functions
type realPcapInterface struct{}

// realPcapHandle wraps the actual pcap.Handle
type realPcapHandle struct {
	handle *pcap.Handle
}

func (r *realPcapInterface) FindAllDevs() ([]pcap.Interface, error) {
	return pcap.FindAllDevs()
}

func (r *realPcapInterface) OpenLive(device string, snaplen int32, promisc bool, timeout time.Duration) (PcapHandle, error) {
	handle, err := pcap.OpenLive(device, snaplen, promisc, timeout)
	if err != nil {
		return nil, err
	}
	return &realPcapHandle{handle: handle}, nil
}

func (h *realPcapHandle) SetBPFFilter(filter string) error {
	return h.handle.SetBPFFilter(filter)
}

func (h *realPcapHandle) Close() {
	h.handle.Close()
}

func (h *realPcapHandle) LinkType() layers.LinkType {
	return h.handle.LinkType()
}

func (h *realPcapHandle) ReadPacketData() ([]byte, gopacket.CaptureInfo, error) {
	return h.handle.ReadPacketData()
}

// pcapOps is the global pcap interface used by the receiver.
// It can be replaced with a mock for testing.
var pcapOps PcapInterface = &realPcapInterface{}
