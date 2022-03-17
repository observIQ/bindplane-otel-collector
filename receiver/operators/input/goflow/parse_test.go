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
	"net"
	"strings"
	"testing"
	"time"

	flowmessage "github.com/observiq/goflow/v3/pb"
	"github.com/stretchr/testify/require"
)

var (
	SamplerAddress = net.IPv4(8, 8, 8, 8)
	SrcAddr        = net.IPv4(1, 1, 1, 1)
	DstAddr        = net.IPv4(10, 1, 1, 1)
	NextHop        = net.IPv4(200, 10, 50, 1)
	SrcAddrEncap   = net.IPv4(1, 10, 2, 1)
	DstAddrEncap   = net.IPv4(10, 10, 10, 10)
)

func TestParse(t *testing.T) {
	cases := []struct {
		name       string
		input      *flowmessage.FlowMessage
		expect     map[string]interface{}
		expectTime time.Time
		expectErr  bool
	}{
		{
			"minimal",
			&flowmessage.FlowMessage{},
			map[string]interface{}{
				"inif":  0,
				"outif": 0,
			},
			time.Time{},
			false,
		},
		{
			"omit-type",
			&flowmessage.FlowMessage{
				Type: flowmessage.FlowMessage_FLOWUNKNOWN,
			},
			map[string]interface{}{
				"inif":  0,
				"outif": 0,
			},
			time.Time{},
			false,
		},
		{
			"type",
			&flowmessage.FlowMessage{
				Type: flowmessage.FlowMessage_IPFIX,
			},
			map[string]interface{}{
				"inif":  0,
				"outif": 0,
				"type":  "IPFIX",
			},
			time.Time{},
			false,
		},
		{
			"addresses",
			&flowmessage.FlowMessage{
				SamplerAddress: SamplerAddress,
				SrcAddr:        SrcAddr,
				DstAddr:        DstAddr,
				NextHop:        NextHop,
			},
			map[string]interface{}{
				"sampleraddress": SamplerAddress.String(),
				"srcaddr":        SrcAddr.String(),
				"dstaddr":        DstAddr.String(),
				"nexthop":        NextHop.String(),
				"inif":           0,
				"outif":          0,
			},
			time.Time{},
			false,
		},
		{
			"empty-srcaddr",
			&flowmessage.FlowMessage{
				SamplerAddress: SamplerAddress,
				SrcAddr:        []byte{},
				DstAddr:        DstAddr,
				NextHop:        NextHop,
			},
			map[string]interface{}{
				"sampleraddress": SamplerAddress.String(),
				"dstaddr":        DstAddr.String(),
				"nexthop":        NextHop.String(),
				"inif":           0,
				"outif":          0,
			},
			time.Time{},
			false,
		},
		{
			"empty-dstaddr",
			&flowmessage.FlowMessage{
				SamplerAddress: SamplerAddress,
				SrcAddr:        SrcAddr,
				DstAddr:        []byte{},
				NextHop:        NextHop,
			},
			map[string]interface{}{
				"sampleraddress": SamplerAddress.String(),
				"srcaddr":        SrcAddr.String(),
				"nexthop":        NextHop.String(),
				"inif":           0,
				"outif":          0,
			},
			time.Time{},
			false,
		},
		{
			"empty-nexthop",
			&flowmessage.FlowMessage{
				SamplerAddress: SamplerAddress,
				SrcAddr:        SrcAddr,
				DstAddr:        DstAddr,
				NextHop:        []byte{},
			},
			map[string]interface{}{
				"sampleraddress": SamplerAddress.String(),
				"srcaddr":        SrcAddr.String(),
				"dstaddr":        DstAddr.String(),
				"inif":           0,
				"outif":          0,
			},
			time.Time{},
			false,
		},
		{
			"empty-srcaddrencap",
			&flowmessage.FlowMessage{
				SamplerAddress: SamplerAddress,
				SrcAddr:        SrcAddr,
				DstAddr:        DstAddr,
				NextHop:        NextHop,
			},
			map[string]interface{}{
				"sampleraddress": SamplerAddress.String(),
				"srcaddr":        SrcAddr.String(),
				"dstaddr":        DstAddr.String(),
				"nexthop":        NextHop.String(),
				"inif":           0,
				"outif":          0,
			},
			time.Time{},
			false,
		},
		{
			"empty-dstaddrencap",
			&flowmessage.FlowMessage{
				SamplerAddress: SamplerAddress,
				SrcAddr:        SrcAddr,
				DstAddr:        DstAddr,
				NextHop:        NextHop,
			},
			map[string]interface{}{
				"sampleraddress": SamplerAddress.String(),
				"srcaddr":        SrcAddr.String(),
				"dstaddr":        DstAddr.String(),
				"nexthop":        NextHop.String(),
				"inif":           0,
				"outif":          0,
			},
			time.Time{},
			false,
		},
		{
			"malformed-addresses",
			&flowmessage.FlowMessage{
				SamplerAddress: SamplerAddress,
				SrcAddr:        []byte("ip:10.1.1.1"),
				DstAddr:        DstAddr,
				NextHop:        NextHop,
			},
			map[string]interface{}{
				"sampleraddress": SamplerAddress.String(),
				"srcaddr":        "ip:10.1.1.1",
				"dstaddr":        DstAddr.String(),
				"nexthop":        NextHop.String(),
				"inif":           0,
				"outif":          0,
			},
			time.Time{},
			true,
		},
		{
			"promote-time",
			&flowmessage.FlowMessage{
				TimeReceived:   1623774351,
				SamplerAddress: SamplerAddress,
				SrcAddr:        SrcAddr,
				DstAddr:        DstAddr,
				NextHop:        NextHop,
			},
			map[string]interface{}{
				"sampleraddress": SamplerAddress.String(),
				"srcaddr":        SrcAddr.String(),
				"dstaddr":        DstAddr.String(),
				"nexthop":        NextHop.String(),
				"inif":           0,
				"outif":          0,
			},
			time.Unix(int64(1623774351), 0),
			false,
		},
		{
			"proto_name",
			&flowmessage.FlowMessage{
				TimeReceived:   1623774351,
				SamplerAddress: SamplerAddress,
				SrcAddr:        SrcAddr,
				DstAddr:        DstAddr,
				NextHop:        NextHop,
				Proto:          1,
			},
			map[string]interface{}{
				"sampleraddress": SamplerAddress.String(),
				"srcaddr":        SrcAddr.String(),
				"dstaddr":        DstAddr.String(),
				"nexthop":        NextHop.String(),
				"proto":          int(1),
				"proto_name":     "ICMP",
				"inif":           0,
				"outif":          0,
			},
			time.Unix(int64(1623774351), 0),
			false,
		},
		{
			"full",
			&flowmessage.FlowMessage{
				TimeReceived:        1623774351,
				SequenceNum:         100,
				SamplingRate:        10,
				FlowDirection:       1,
				SamplerAddress:      SamplerAddress,
				TimeFlowStart:       1623774351,
				TimeFlowEnd:         1623774351,
				Bytes:               100,
				Packets:             200,
				SrcAddr:             SrcAddr,
				DstAddr:             DstAddr,
				Etype:               40,
				Proto:               40,
				SrcPort:             2,
				DstPort:             40,
				InIf:                10,
				OutIf:               1,
				SrcMac:              100000000,
				DstMac:              100000000,
				SrcVlan:             10,
				DstVlan:             20,
				VlanId:              4,
				IngressVrfID:        4,
				EgressVrfID:         0,
				IPTos:               1,
				ForwardingStatus:    4,
				IPTTL:               5,
				TCPFlags:            5,
				IcmpType:            5,
				IcmpCode:            5,
				IPv6FlowLabel:       5,
				FragmentId:          5,
				FragmentOffset:      5,
				BiFlowDirection:     5,
				SrcAS:               5,
				DstAS:               5,
				NextHop:             SamplerAddress,
				NextHopAS:           5,
				SrcNet:              24,
				DstNet:              16,
				HasEncap:            true,
				SrcAddrEncap:        SrcAddrEncap,
				DstAddrEncap:        DstAddrEncap,
				ProtoEncap:          5,
				EtypeEncap:          5,
				IPTosEncap:          5,
				IPTTLEncap:          5,
				IPv6FlowLabelEncap:  5,
				FragmentIdEncap:     5,
				FragmentOffsetEncap: 5,
				HasMPLS:             true,
				MPLSCount:           5,
				MPLS1TTL:            5,
				MPLS1Label:          5,
				MPLS2TTL:            5,
				MPLS2Label:          5,
				MPLS3TTL:            5,
				MPLS3Label:          5,
				MPLSLastTTL:         5,
				MPLSLastLabel:       5,
				HasPPP:              false,
				PPPAddressControl:   0,
			},
			map[string]interface{}{
				"bytes":               int64(100),
				"dstaddr":             "10.1.1.1",
				"dstaddrencap":        "10.10.10.10",
				"dstmac":              int64(100000000),
				"dstnas":              5,
				"dstnet":              16,
				"dstport":             40,
				"dstvlan":             20,
				"etype":               40,
				"etypeencap":          5,
				"forwardingstatus":    4,
				"fragmentid":          5,
				"fragmentidencap":     5,
				"fragmentoffset":      5,
				"fragmentoffsetencap": 5,
				"icmpcode":            5,
				"icmptype":            5,
				"ingressvrfid":        4,
				"iptos":               1,
				"iptosencap":          5,
				"ipttl":               5,
				"ipttlencap":          5,
				"ipv6flowlabel":       5,
				"ipv6flowlabelencap":  5,
				"mpls1label":          5,
				"mpls1ttl":            5,
				"mpls2label":          5,
				"mpls2ttl":            5,
				"mpls3label":          5,
				"mpls3ttl":            5,
				"mplscount":           5,
				"mplslastlabel":       5,
				"mplslastttl":         5,
				"nexthop":             "8.8.8.8",
				"nexthopas":           5,
				"packets":             int64(200),
				"protoencap":          5,
				"sampleraddress":      "8.8.8.8",
				"samplingrate":        int64(10),
				"sequencenum":         100,
				"srcaddr":             "1.1.1.1",
				"srcaddrencap":        "1.10.2.1",
				"srcas":               5,
				"srcmac":              int64(100000000),
				"srcnet":              24,
				"srcport":             2,
				"srcvlan":             10,
				"tcpflags":            5,
				"timeflowend":         int64(1623774351),
				"timeflowstart":       int64(1623774351),
				"vlanid":              4,
				"proto":               40,
				"proto_name":          "IL",
				"inif":                10,
				"outif":               1,
			},
			time.Unix(int64(1623774351), 0),
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, outputTime, err := Parse(tc.input)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expect, output)

			if tc.input.TimeReceived > 0 {
				require.Equal(t, tc.expectTime, outputTime, "expected field timereceived to be promoted to timestamp")
			}

			for k := range output {
				require.Equal(t, strings.ToLower(k), k, "expected all keys to be lowercase")
			}
		})
	}
}

func BenchmarkParse(b *testing.B) {
	m := &flowmessage.FlowMessage{
		Type:                flowmessage.FlowMessage_NETFLOW_V5,
		TimeReceived:        uint64(time.Nanosecond),
		SequenceNum:         100,
		SamplingRate:        10,
		FlowDirection:       1,
		SamplerAddress:      SamplerAddress,
		TimeFlowStart:       uint64(time.Nanosecond),
		TimeFlowEnd:         uint64(time.Nanosecond),
		Bytes:               100,
		Packets:             200,
		SrcAddr:             SrcAddr,
		DstAddr:             DstAddr,
		Etype:               40,
		Proto:               40,
		SrcPort:             2,
		DstPort:             40,
		InIf:                10,
		OutIf:               1,
		SrcMac:              100000000,
		DstMac:              100000000,
		SrcVlan:             10,
		DstVlan:             20,
		VlanId:              4,
		IngressVrfID:        4,
		EgressVrfID:         0,
		IPTos:               1,
		ForwardingStatus:    4,
		IPTTL:               5,
		TCPFlags:            5,
		IcmpType:            5,
		IcmpCode:            5,
		IPv6FlowLabel:       5,
		FragmentId:          5,
		FragmentOffset:      5,
		BiFlowDirection:     5,
		SrcAS:               5,
		DstAS:               5,
		NextHop:             SamplerAddress,
		NextHopAS:           5,
		SrcNet:              24,
		DstNet:              16,
		HasEncap:            true,
		SrcAddrEncap:        SrcAddrEncap,
		DstAddrEncap:        DstAddrEncap,
		ProtoEncap:          5,
		EtypeEncap:          5,
		IPTosEncap:          5,
		IPTTLEncap:          5,
		IPv6FlowLabelEncap:  5,
		FragmentIdEncap:     5,
		FragmentOffsetEncap: 5,
		HasMPLS:             true,
		MPLSCount:           5,
		MPLS1TTL:            5,
		MPLS1Label:          5,
		MPLS2TTL:            5,
		MPLS2Label:          5,
		MPLS3TTL:            5,
		MPLS3Label:          5,
		MPLSLastTTL:         5,
		MPLSLastLabel:       5,
		HasPPP:              false,
		PPPAddressControl:   0,
	}

	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		_, _, err := Parse(m)
		if err != nil {
			b.Errorf(err.Error())
			b.FailNow()
		}
	}
}
