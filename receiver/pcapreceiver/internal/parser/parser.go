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

package parser

import (
	"github.com/google/gopacket"
	"go.opentelemetry.io/collector/pdata/plog"
)

// Parser converts gopacket.Packet to OpenTelemetry logs
type Parser struct {
	interfaceName string
}

// NewParser creates a new packet parser
func NewParser(interfaceName string) *Parser {
	return &Parser{
		interfaceName: interfaceName,
	}
}

// Parse converts a packet to plog.Logs
func (p *Parser) Parse(packet gopacket.Packet) plog.Logs {
	// To be implemented in Phase 6
	return plog.NewLogs()
}

