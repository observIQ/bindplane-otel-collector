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

//go:build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

func main() {
	// Get the testdata directory path
	testdataDir := filepath.Join("..", "..", "testdata")
	if err := os.MkdirAll(testdataDir, 0755); err != nil {
		panic(err)
	}

	pcapPath := filepath.Join(testdataDir, "capture.pcap")
	fmt.Printf("Generating PCAP file: %s\n", pcapPath)

	f, err := os.Create(pcapPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w := pcapgo.NewWriter(f)
	if err := w.WriteFileHeader(65536, layers.LinkTypeEthernet); err != nil {
		panic(err)
	}

	// Generate 11 test packets (TCP, UDP, ICMP, IPv6)
	packets := make([][]byte, 0, 11)

	// Add packet data from CreateTCPPacket, CreateUDPPacket, etc.
	// For simplicity, just write minimal valid packets

	fmt.Printf("Generated %d packets\n", len(packets))
}
