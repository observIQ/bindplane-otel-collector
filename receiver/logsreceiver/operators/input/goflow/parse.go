// Copyright  observIQ, Inc.
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

package goflow

import (
	"fmt"
	"net"
	"time"

	flowmessage "github.com/observiq/goflow/v3/pb"
	"github.com/open-telemetry/opentelemetry-log-collection/errors"
)

// Parse parses a netflow message into an entry. It is assumed that Proto,
// InIf, and OutIf are always set because 0 values are valid.
func Parse(message *flowmessage.FlowMessage) (map[string]interface{}, time.Time, error) {
	m := make(map[string]interface{})

	timestamp := time.Unix(int64(message.TimeReceived), 0)

	if t := message.Type.String(); t != "" {
		if t != "FLOWUNKNOWN" {
			m["type"] = t
		}
	}

	if message.SequenceNum > 0 {
		m["sequencenum"] = int(message.SequenceNum)
	}

	if message.SamplingRate > 0 {
		m["samplingrate"] = int64(message.SamplingRate)
	}

	if message.HasFlowDirection {
		m["flowdirection"] = int(message.FlowDirection)
	}

	if len(message.SamplerAddress) > 0 {
		key := "sampleraddress"
		ip, err := bytesToIP(message.SamplerAddress)
		if err != nil {
			return nil, timestamp, errors.Wrap(err, fmt.Sprintf("error converting %s to string", key))
		}
		m[key] = ip.String()
	}

	if message.TimeFlowStart > 0 {
		m["timeflowstart"] = int64(message.TimeFlowStart)
	}

	if message.TimeFlowEnd > 0 {
		m["timeflowend"] = int64(message.TimeFlowEnd)
	}

	if message.Bytes > 0 {
		m["bytes"] = int64(message.Bytes)
	}

	if message.Packets > 0 {
		m["packets"] = int64(message.Packets)
	}

	if len(message.SrcAddr) > 0 {
		key := "srcaddr"
		ip, err := bytesToIP(message.SrcAddr)
		if err != nil {
			return nil, timestamp, errors.Wrap(err, fmt.Sprintf("error converting %s to string", key))
		}
		m[key] = ip.String()
	}

	if len(message.DstAddr) > 0 {
		key := "dstaddr"
		ip, err := bytesToIP(message.DstAddr)
		if err != nil {
			return nil, timestamp, errors.Wrap(err, fmt.Sprintf("error converting %s to string", key))
		}
		m[key] = ip.String()
	}

	if len(message.NextHop) > 0 {
		key := "nexthop"
		ip, err := bytesToIP(message.NextHop)
		if err != nil {
			return nil, timestamp, errors.Wrap(err, fmt.Sprintf("error converting %s to string", key))
		}
		m[key] = ip.String()
	}

	if message.Etype > 0 {
		m["etype"] = int(message.Etype)
	}

	// Goflow input does not support HOPOPT as it maps to 0, meaning HOPOPT would
	// be set anytime the proto field is not present.
	if message.Proto > 0 {
		m["proto"] = int(message.Proto)
		m["proto_name"] = protoName(int(message.Proto))
	}

	if message.SrcPort > 0 {
		m["srcport"] = int(message.SrcPort)
	}

	if message.DstPort > 0 {
		m["dstport"] = int(message.DstPort)
	}

	// Always set inif and outif
	m["inif"] = int(message.InIf)
	m["outif"] = int(message.OutIf)

	if message.SrcMac > 0 {
		m["srcmac"] = int64(message.SrcMac)
	}

	if message.DstMac > 0 {
		m["dstmac"] = int64(message.DstMac)
	}

	if message.SrcVlan > 0 {
		m["srcvlan"] = int(message.SrcVlan)
	}

	if message.DstVlan > 0 {
		m["dstvlan"] = int(message.DstVlan)
	}

	if message.VlanId > 0 {
		m["vlanid"] = int(message.VlanId)
	}

	if message.IngressVrfID > 0 {
		m["ingressvrfid"] = int(message.IngressVrfID)
	}

	if message.EgressVrfID > 0 {
		m["egressvrfid"] = int(message.EgressVrfID)
	}

	if message.IPTos > 0 {
		m["iptos"] = int(message.IPTos)
	}

	if message.ForwardingStatus > 0 {
		m["forwardingstatus"] = int(message.ForwardingStatus)
	}

	if message.IPTTL > 0 {
		m["ipttl"] = int(message.IPTTL)
	}

	if message.TCPFlags > 0 {
		m["tcpflags"] = int(message.TCPFlags)
	}

	if message.IcmpType > 0 {
		m["icmptype"] = int(message.IcmpType)
	}

	if message.IcmpCode > 0 {
		m["icmpcode"] = int(message.IcmpCode)
	}

	if message.IPv6FlowLabel > 0 {
		m["ipv6flowlabel"] = int(message.IPv6FlowLabel)
	}

	if message.FragmentId > 0 {
		m["fragmentid"] = int(message.FragmentId)
	}

	if message.FragmentOffset > 0 {
		m["fragmentoffset"] = int(message.FragmentOffset)
	}

	if message.HasBiFlowDirection {
		m["biflowdirection"] = int(message.BiFlowDirection)
	}

	if message.SrcAS > 0 {
		m["srcas"] = int(message.SrcAS)
	}

	if message.DstAS > 0 {
		m["dstnas"] = int(message.DstAS)
	}

	if message.NextHopAS > 0 {
		m["nexthopas"] = int(message.NextHopAS)
	}

	if message.SrcNet > 0 {
		m["srcnet"] = int(message.SrcNet)
	}

	if message.DstNet > 0 {
		m["dstnet"] = int(message.DstNet)
	}

	// Add Encap fields if present
	if message.HasEncap {
		key := "srcaddrencap"
		ip, err := bytesToIP(message.SrcAddrEncap)
		if err != nil {
			return nil, timestamp, errors.Wrap(err, fmt.Sprintf("error converting %s to string", key))
		}
		m[key] = ip.String()

		key = "dstaddrencap"
		ip, err = bytesToIP(message.DstAddrEncap)
		if err != nil {
			return nil, timestamp, errors.Wrap(err, fmt.Sprintf("error converting %s to string", key))
		}
		m[key] = ip.String()

		m["protoencap"] = int(message.ProtoEncap)
		m["etypeencap"] = int(message.EtypeEncap)
		m["iptosencap"] = int(message.IPTosEncap)
		m["ipttlencap"] = int(message.IPTTLEncap)
		m["ipv6flowlabelencap"] = int(message.IPv6FlowLabelEncap)
		m["fragmentidencap"] = int(message.FragmentIdEncap)
		m["fragmentoffsetencap"] = int(message.FragmentOffsetEncap)
	}

	// Add MPLS fields if present
	if message.HasMPLS {
		m["mplscount"] = int(message.MPLSCount)
		m["mpls1ttl"] = int(message.MPLS1TTL)
		m["mpls1label"] = int(message.MPLS1Label)
		m["mpls2ttl"] = int(message.MPLS2TTL)
		m["mpls2label"] = int(message.MPLS2Label)
		m["mpls3ttl"] = int(message.MPLS3TTL)
		m["mpls3label"] = int(message.MPLS3Label)
		m["mplslastttl"] = int(message.MPLSLastTTL)
		m["mplslastlabel"] = int(message.MPLSLastLabel)
	}

	// Add PPP fields if present
	if message.HasPPP {
		m["sampling_rate"] = int(message.PPPAddressControl)
	}

	return m, timestamp, nil
}

// converts []byte to ip address
func bytesToIP(b []byte) (net.IP, error) {
	switch x := len(b); x {
	case 4, 16:
		var ip net.IP = b
		return ip, nil
	default:
		return nil, fmt.Errorf("cannot convert byte slice to ip address, expected length of 4 or 16 got %d", x)
	}
}
