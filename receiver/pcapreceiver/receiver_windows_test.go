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
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
)

func TestCheckPrivileges_Windows_NpcapCheck(t *testing.T) {
	cfg := &Config{Interface: `\Device\NPF_{12345678-1234-1234-1234-123456789012}`}
	receiver := newTestReceiver(t, cfg, nil, nil)

	// This will try to actually open the interface using go-pcap
	// It may fail if Npcap is not installed or interface doesn't exist
	err := receiver.checkPrivileges()
	// If Npcap is available and interface exists, should succeed
	// Otherwise, should fail with Npcap error
	if err != nil {
		require.Contains(t, err.Error(), "Npcap", "Error should mention Npcap: %v", err.Error())
	}
}

func TestStart_InvalidConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError string
	}{
		{
			name: "empty interface",
			config: &Config{
				Interface: "",
			},
			wantError: "interface must be specified",
		},
		{
			name: "invalid interface with shell injection",
			config: &Config{
				Interface: "1; whoami",
			},
			wantError: "invalid character",
		},
		{
			name: "invalid filter",
			config: &Config{
				Interface: `\Device\NPF_{12345678-1234-1234-1234-123456789012}`,
				Filter:    "tcp port 80 && whoami",
			},
			wantError: "invalid character",
		},
		{
			name: "invalid snaplen",
			config: &Config{
				Interface: `\Device\NPF_{12345678-1234-1234-1234-123456789012}`,
				SnapLen:   10,
			},
			wantError: "snaplen must be between",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiver := newTestReceiver(t, tt.config, nil, nil)

			err := receiver.Start(context.Background(), componenttest.NewNopHost())
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantError)
		})
	}
}
