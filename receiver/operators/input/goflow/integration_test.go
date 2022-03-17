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

//go:build integration

package goflow

import (
	"bufio"
	"context"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	flowmessage "github.com/observiq/goflow/v3/pb"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

func netFlowV5GenContainer(targetIP string) (testcontainers.Container, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:      "bindplane/nflow-generator:1.0.0",
		Entrypoint: []string{"/go/bin/nflow-generator", "-t", targetIP, "-p", "2056"},
		BindMounts: map[string]string{
			path.Join(workingDir, "testdata"): "/testdata",
		},
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

func stanzaContainer() (testcontainers.Container, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "stanza-integration:latest",
		ExposedPorts: []string{"2056:2056"},
		Entrypoint:   []string{"/bin/sleep", "9999"},
		BindMounts: map[string]string{
			path.Join(workingDir, "testdata"): "/testdata",
		},
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
}

func TestNetflowV5(t *testing.T) {
	stanza, err := stanzaContainer()
	if err != nil {
		require.NoError(t, err)
		return
	}
	defer func() {
		err := stanza.Terminate(context.Background())
		require.NoError(t, err, "failed to cleanup test container")
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := stanza.Exec(ctx, []string{"sh", "/testdata/netflowv5.sh"}); err != nil {
		if !strings.Contains(err.Error(), "context deadline exceeded") {
			require.NoError(t, err)
			return
		}
	}

	stanzaIP, err := stanza.ContainerIP(context.Background())
	if err != nil {
		require.NoError(t, err, "expected nil error when getting stanza container ip")
		return
	}

	loadgen, err := netFlowV5GenContainer(stanzaIP)
	if err != nil {
		require.NoError(t, err)
		return
	}
	defer func() {
		err := loadgen.Terminate(context.Background())
		require.NoError(t, err, "failed to cleanup test container")
	}()

	time.Sleep(time.Second * 2)

	// Read entire files into memory. Don't do this if you expect the output to be massive.
	// These files are reset to 0 lines on every run in testdata/netflowv5.sh
	//
	// Expect stanza stdout / stderr to contain zero output
	b, err := ioutil.ReadFile("./testdata/stdout.log")
	if err != nil {
		require.NoError(t, err, "expected to read testdata/stdout.log")
		return
	}
	require.Equal(t, 0, len(b), "expected stdout.log to be empty, this indicates the agent paniced or logged an error")

	// Expect stanza's log to contain exactly 3 lines
	/*
		{"level":"info","timestamp":"2021-06-15T21:03:10.763Z","message":"Starting stanza agent"}
		{"level":"info","timestamp":"2021-06-15T21:03:10.763Z","message":"Started Goflow on 0.0.0.0:2056 in netflow_v5 mode","operator_id":"$.goflow_input","operator_type":"goflow_input"}
		{"level":"info","timestamp":"2021-06-15T21:03:10.763Z","message":"Stanza agent started"}
		< new line >
	*/
	b, err = ioutil.ReadFile("./testdata/stanza.log") // just pass the file name
	if err != nil {
		require.NoError(t, err, "expected to read testdata/stanza.log")
		return
	}
	require.Equal(t, 4, len(strings.Split(string(b), "\n")), "expected stanza.log to contain exactly 3 lines")

	// Test file_output contents
	samplerAddress, err := loadgen.ContainerIP(context.Background())
	if err != nil {
		require.NoError(t, err, "expected nil error when getting stanza container ip")
		return
	}

	f, err := os.Open("./testdata/out.log")
	if err != nil {
		require.NoError(t, err)
		return
	}
	defer f.Close()
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		m := OutputEntry{}
		if err := json.Unmarshal(scanner.Bytes(), &m); err != nil {
			require.NoError(t, err, "expected to unmarshal stanza file output to FlowMessage struct")
			return
		}
		require.NotEmpty(t, m.Record)
		require.NotEqual(t, m.Timestamp, time.Time{})
		require.Equal(t, samplerAddress, m.Record.SamplerAddress, "expected sampleraddress to be the loadgen container's ip address")
		return
	}
	if err := scanner.Err(); err != nil {
		require.NoError(t, err)
		return
	}
}

type OutputEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Severity  int       `json:"severity"`
	Record    struct {
		// based on https://github.com/cloudflare/goflow/blob/ddd88a7faa89bd9a8e75f0ceca17cbb443c14a8f/pb/flow.pb.go
		// all ip addresses are strings instead of []byte, and noted with a comment
		Type          flowmessage.FlowMessage_FlowType `json:"type,omitempty"`
		TimeReceived  uint64                           `json:"timereceived,omitempty"`
		SequenceNum   uint32                           `json:"sequencenum,omitempty"`
		SamplingRate  uint64                           `json:"samplingrate,omitempty"`
		FlowDirection uint32                           `json:"flowdirection,omitempty"`
		// converted to string
		SamplerAddress string `json:"sampleraddress,omitempty"`
		TimeFlowStart  uint64 `json:"timeflowstart,omitempty"`
		TimeFlowEnd    uint64 `json:"timeflowend,omitempty"`
		Bytes          uint64 `json:"bytes,omitempty"`
		Packets        uint64 `json:"packets,omitempty"`
		// converted to string
		SrcAddr string `json:"srcaddr,omitempty"`
		// converted to string
		DstAddr          string `json:"dstaddr,omitempty"`
		Etype            uint32 `json:"etype,omitempty"`
		Proto            uint32 `json:"proto,omitempty"`
		SrcPort          uint32 `json:"srcport,omitempty"`
		DstPort          uint32 `json:"dstport,omitempty"`
		InIf             uint32 `json:"inif,omitempty"`
		OutIf            uint32 `json:"outif,omitempty"`
		SrcMac           uint64 `json:"srcmac,omitempty"`
		DstMac           uint64 `json:"dstmac,omitempty"`
		SrcVlan          uint32 `json:"srcvlan,omitempty"`
		DstVlan          uint32 `json:"dstvlan,omitempty"`
		VlanId           uint32 `json:"vlanid,omitempty"`
		IngressVrfID     uint32 `json:"ingressvrfid,omitempty"`
		EgressVrfID      uint32 `json:"egressvrfid,omitempty"`
		IPTos            uint32 `json:"iptos,omitempty"`
		ForwardingStatus uint32 `json:"forwardingstatus,omitempty"`
		IPTTL            uint32 `json:"ipttl,omitempty"`
		TCPFlags         uint32 `json:"tcpflags,omitempty"`
		IcmpType         uint32 `json:"icmptype,omitempty"`
		IcmpCode         uint32 `json:"icmpcode,omitempty"`
		IPv6FlowLabel    uint32 `json:"ipv6flowlabel,omitempty"`
		FragmentId       uint32 `json:"fragmentid,omitempty"`
		FragmentOffset   uint32 `json:"fragmentoffset,omitempty"`
		BiFlowDirection  uint32 `json:"biflowdirection,omitempty"`
		SrcAS            uint32 `json:"srcas,omitempty"`
		DstAS            uint32 `json:"dstas,omitempty"`
		// converted to string
		NextHop   string `json:"nexthop,omitempty"`
		NextHopAS uint32 `json:"nexthopas,omitempty"`
		SrcNet    uint32 `json:"srcnet,omitempty"`
		DstNet    uint32 `json:"dstnet,omitempty"`
		HasEncap  bool   `json:"hasencap,omitempty"`
		// converted to string
		SrcAddrEncap string `json:"srcaddrencap,omitempty"`
		// converted to string
		DstAddrEncap         string   `json:"dstaddrencap,omitempty"`
		ProtoEncap           uint32   `json:"protoencap,omitempty"`
		EtypeEncap           uint32   `json:"etypeencap,omitempty"`
		IPTosEncap           uint32   `json:"iptosencap,omitempty"`
		IPTTLEncap           uint32   `json:"ipttlencap,omitempty"`
		IPv6FlowLabelEncap   uint32   `json:"ipv6flowlabelencap,omitempty"`
		FragmentIdEncap      uint32   `json:"fragmentidencap,omitempty"`
		FragmentOffsetEncap  uint32   `json:"fragmentoffsetencap,omitempty"`
		HasMPLS              bool     `json:"hasmpls,omitempty"`
		MPLSCount            uint32   `json:"mplscount,omitempty"`
		MPLS1TTL             uint32   `json:"mpls1ttl,omitempty"`
		MPLS1Label           uint32   `json:"mpls1label,omitempty"`
		MPLS2TTL             uint32   `json:"mpls2ttl,omitempty"`
		MPLS2Label           uint32   `json:"mpls2label,omitempty"`
		MPLS3TTL             uint32   `json:"mpls3ttl,omitempty"`
		MPLS3Label           uint32   `json:"mpls3label,omitempty"`
		MPLSLastTTL          uint32   `json:"mplslastttl,omitempty"`
		MPLSLastLabel        uint32   `json:"mplslastlabel,omitempty"`
		HasPPP               bool     `json:"hasppp,omitempty"`
		PPPAddressControl    uint32   `json:"pppaddresscontrol,omitempty"`
		XXX_NoUnkeyedLiteral struct{} `json:"-"`
		XXX_unrecognized     []byte   `json:"-"`
		XXX_sizecache        int32    `json:"-"`
	} `json:"record"`
}
