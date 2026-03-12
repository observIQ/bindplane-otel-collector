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
	"errors"
	"io"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
)

// mockPcapInterface is a mock implementation of PcapInterface for testing
type mockPcapInterface struct {
	devices    []pcap.Interface
	findAllErr error
	openHandle PcapHandle
	openErr    error
}

func (m *mockPcapInterface) FindAllDevs() ([]pcap.Interface, error) {
	if m.findAllErr != nil {
		return nil, m.findAllErr
	}
	return m.devices, nil
}

func (m *mockPcapInterface) OpenLive(_ string, _ int32, _ bool, _ time.Duration) (PcapHandle, error) {
	if m.openErr != nil {
		return nil, m.openErr
	}
	return m.openHandle, nil
}

// mockPcapHandle is a mock implementation of PcapHandle for testing
type mockPcapHandle struct {
	packets      []mockPacket
	packetIndex  int
	setBPFErr    error
	closed       bool
	linkType     layers.LinkType
	readPacketFn func() ([]byte, gopacket.CaptureInfo, error)
}

type mockPacket struct {
	data []byte
	ci   gopacket.CaptureInfo
	err  error
}

func newMockPcapHandle() *mockPcapHandle {
	return &mockPcapHandle{
		linkType: layers.LinkTypeEthernet,
	}
}

func (h *mockPcapHandle) SetBPFFilter(_ string) error {
	return h.setBPFErr
}

func (h *mockPcapHandle) Close() {
	h.closed = true
}

func (h *mockPcapHandle) LinkType() layers.LinkType {
	return h.linkType
}

func (h *mockPcapHandle) ReadPacketData() ([]byte, gopacket.CaptureInfo, error) {
	if h.readPacketFn != nil {
		return h.readPacketFn()
	}
	if h.packetIndex >= len(h.packets) {
		return nil, gopacket.CaptureInfo{}, io.EOF
	}
	pkt := h.packets[h.packetIndex]
	h.packetIndex++
	return pkt.data, pkt.ci, pkt.err
}

// withDevices configures the mock to return the given devices
func (m *mockPcapInterface) withDevices(devices ...pcap.Interface) *mockPcapInterface {
	m.devices = devices
	return m
}

// withFindAllError configures the mock to return an error from FindAllDevs
func (m *mockPcapInterface) withFindAllError(err error) *mockPcapInterface {
	m.findAllErr = err
	return m
}

// withOpenHandle configures the mock to return the given handle from OpenLive
func (m *mockPcapInterface) withOpenHandle(handle PcapHandle) *mockPcapInterface {
	m.openHandle = handle
	return m
}

// withOpenError configures the mock to return an error from OpenLive
func (m *mockPcapInterface) withOpenError(err error) *mockPcapInterface {
	m.openErr = err
	return m
}

// withSetBPFError configures the mock to return an error from SetBPFFilter
func (h *mockPcapHandle) withSetBPFError(err error) *mockPcapHandle {
	h.setBPFErr = err
	return h
}

// withPackets configures the mock to return the given packets from ReadPacketData
func (h *mockPcapHandle) withPackets(packets ...mockPacket) *mockPcapHandle {
	h.packets = packets
	return h
}

// newMockPcapInterface creates a new mock pcap interface
func newMockPcapInterface() *mockPcapInterface {
	return &mockPcapInterface{}
}

// setupMock temporarily replaces the global pcapOps with the provided mock
// and returns a cleanup function to restore the original.
//
// NOTE: Tests using setupMock must NOT use t.Parallel() as they modify
// the global pcapOps variable. Running such tests in parallel would cause
// race conditions and unpredictable behavior.
func setupMock(mock PcapInterface) func() {
	original := pcapOps
	pcapOps = mock
	return func() {
		pcapOps = original
	}
}

// errNpcapNotInstalled simulates the Npcap not installed error
var errNpcapNotInstalled = errors.New("wpcap.dll not found")

// Sample TCP packet bytes (Ethernet + IPv4 + TCP SYN)
// This is a minimal valid packet for testing the packet processing path
var sampleTCPPacket = []byte{
	// Ethernet header (14 bytes)
	0x00, 0x11, 0x22, 0x33, 0x44, 0x55, // Dst MAC
	0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, // Src MAC
	0x08, 0x00, // EtherType: IPv4
	// IPv4 header (20 bytes)
	0x45, 0x00, 0x00, 0x28, // Version/IHL, DSCP/ECN, Total Length (40 bytes)
	0x00, 0x01, 0x00, 0x00, // ID, Flags/Fragment
	0x40, 0x06, 0x00, 0x00, // TTL, Protocol (TCP), Checksum
	0xc0, 0xa8, 0x01, 0x64, // Src IP: 192.168.1.100
	0xc0, 0xa8, 0x01, 0x01, // Dst IP: 192.168.1.1
	// TCP header (20 bytes)
	0xd4, 0x31, // Src Port: 54321
	0x01, 0xbb, // Dst Port: 443
	0x00, 0x00, 0x00, 0x01, // Seq number
	0x00, 0x00, 0x00, 0x00, // Ack number
	0x50, 0x02, 0xff, 0xff, // Data offset, Flags (SYN), Window
	0x00, 0x00, 0x00, 0x00, // Checksum, Urgent pointer
}

// Sample UDP packet bytes (Ethernet + IPv4 + UDP)
var sampleUDPPacket = []byte{
	// Ethernet header (14 bytes)
	0x00, 0x11, 0x22, 0x33, 0x44, 0x55, // Dst MAC
	0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, // Src MAC
	0x08, 0x00, // EtherType: IPv4
	// IPv4 header (20 bytes)
	0x45, 0x00, 0x00, 0x1c, // Version/IHL, DSCP/ECN, Total Length
	0x00, 0x01, 0x00, 0x00, // ID, Flags/Fragment
	0x40, 0x11, 0x00, 0x00, // TTL, Protocol (UDP), Checksum
	0x0a, 0x00, 0x00, 0x05, // Src IP: 10.0.0.5
	0x08, 0x08, 0x08, 0x08, // Dst IP: 8.8.8.8
	// UDP header (8 bytes)
	0x30, 0x39, // Src Port: 12345
	0x00, 0x35, // Dst Port: 53
	0x00, 0x08, 0x00, 0x00, // Length, Checksum
}
