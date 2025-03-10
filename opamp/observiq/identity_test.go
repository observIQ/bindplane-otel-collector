// Copyright  Bindplane, Inc.
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

package observiq

import (
	"os"
	"runtime"
	"testing"

	"github.com/observiq/bindplane-otel-collector/opamp"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Must is a helper function for tests that panics if there is an error creating the object of type T
func Must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

var testAgentID = Must(opamp.ParseAgentID("01HX2DWEQZ045KQR3VG0EYEZ94"))

func Test_newIdentity(t *testing.T) {
	secretKeyContents := "b92222ee-a1fc-4bb1-98db-26de3448541b"
	labelsContents := "one=foo,two=bar"
	agentNameContents := "My Agent"

	cfg := opamp.Config{
		Endpoint:  "ws://localhost:1234",
		SecretKey: &secretKeyContents,
		AgentID:   testAgentID,
		Labels:    &labelsContents,
		AgentName: &agentNameContents,
	}

	expectedVersion := "0.0.0"

	got := newIdentity(zap.NewNop(), cfg, expectedVersion)

	// Check all fields from config
	require.Equal(t, cfg.AgentID, got.agentID)
	require.Equal(t, cfg.AgentName, got.agentName)
	require.Equal(t, cfg.Labels, got.labels)

	// Check fields that must not be empty
	require.NotEmpty(t, got.oSDetails)
	require.NotEmpty(t, got.hostname)
	require.NotEmpty(t, got.mac)

	// Check hardcoded/fields from runtime and other packages
	require.Equal(t, got.serviceName, "com.observiq.collector")
	require.Equal(t, got.version, expectedVersion)
	require.Equal(t, got.oSArch, runtime.GOARCH)
	require.Equal(t, got.oSFamily, runtime.GOOS)
}

func TestToAgentDescription(t *testing.T) {
	labelsContents := "one=foo,two=bar"
	agentNameContents := "My Agent"
	testCases := []struct {
		desc     string
		ident    *identity
		expected *protobufs.AgentDescription
	}{
		{
			desc: "Missing Agent Name and labels",
			ident: &identity{
				agentID:     testAgentID,
				agentName:   nil,
				serviceName: "com.observiq.collector",
				version:     "v1.2.3",
				labels:      nil,
				oSArch:      "amd64",
				oSDetails:   "os details",
				oSFamily:    "linux",
				hostname:    "my-linux-box",
				mac:         "68-C7-B4-EB-A8-D2",
			},
			expected: &protobufs.AgentDescription{
				IdentifyingAttributes: []*protobufs.KeyValue{
					opamp.StringKeyValue("service.instance.id", testAgentID.String()),
					opamp.StringKeyValue("service.name", "com.observiq.collector"),
					opamp.StringKeyValue("service.version", "v1.2.3"),
					opamp.StringKeyValue("service.instance.name", "my-linux-box"),
				},
				NonIdentifyingAttributes: []*protobufs.KeyValue{
					opamp.StringKeyValue("os.arch", "amd64"),
					opamp.StringKeyValue("os.details", "os details"),
					opamp.StringKeyValue("os.family", "linux"),
					opamp.StringKeyValue("host.name", "my-linux-box"),
					opamp.StringKeyValue("host.mac_address", "68-C7-B4-EB-A8-D2"),
					opamp.StringKeyValue("service.key", "test-key"),
				},
			},
		},
		{
			desc: "With Agent Name and labels",
			ident: &identity{
				agentID:     testAgentID,
				agentName:   &agentNameContents,
				serviceName: "com.observiq.collector",
				version:     "v1.2.3",
				labels:      &labelsContents,
				oSArch:      "amd64",
				oSDetails:   "os details",
				oSFamily:    "linux",
				hostname:    "my-linux-box",
				mac:         "68-C7-B4-EB-A8-D2",
			},
			expected: &protobufs.AgentDescription{
				IdentifyingAttributes: []*protobufs.KeyValue{
					opamp.StringKeyValue("service.instance.id", testAgentID.String()),
					opamp.StringKeyValue("service.name", "com.observiq.collector"),
					opamp.StringKeyValue("service.version", "v1.2.3"),
					opamp.StringKeyValue("service.instance.name", agentNameContents),
				},
				NonIdentifyingAttributes: []*protobufs.KeyValue{
					opamp.StringKeyValue("os.arch", "amd64"),
					opamp.StringKeyValue("os.details", "os details"),
					opamp.StringKeyValue("os.family", "linux"),
					opamp.StringKeyValue("host.name", "my-linux-box"),
					opamp.StringKeyValue("host.mac_address", "68-C7-B4-EB-A8-D2"),
					opamp.StringKeyValue("service.labels", labelsContents),
					opamp.StringKeyValue("service.key", "test-key"),
				},
			},
		},
	}

	for _, tc := range testCases {
		os.Setenv("OTEL_AES_CREDENTIAL_PROVIDER", "test-key")
		t.Run(tc.desc, func(t *testing.T) {
			actual := tc.ident.ToAgentDescription()
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func Test_identityCopy(t *testing.T) {
	labelsContents := "one=foo,two=bar"
	agentNameContents := "My Agent"
	ident := &identity{
		agentID:     testAgentID,
		agentName:   &agentNameContents,
		serviceName: "com.observiq.collector",
		version:     "v1.2.3",
		labels:      &labelsContents,
		oSArch:      "amd64",
		oSDetails:   "os details",
		oSFamily:    "linux",
		hostname:    "my-linux-box",
		mac:         "68-C7-B4-EB-A8-D2",
	}

	copyIdent := ident.Copy()

	require.Equal(t, ident, copyIdent)
}
