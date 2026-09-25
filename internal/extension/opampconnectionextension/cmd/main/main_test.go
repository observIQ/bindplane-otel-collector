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

package main

import (
	"testing"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/require"
)

func codeNamespace(ref string) *protobufs.KeyValue {
	return &protobufs.KeyValue{
		Key:   "code.namespace",
		Value: &protobufs.AnyValue{Value: &protobufs.AnyValue_StringValue{StringValue: ref}},
	}
}

func values(md []*protobufs.KeyValue) []string {
	out := make([]string, 0, len(md))
	for _, kv := range md {
		out = append(out, kv.GetValue().GetStringValue())
	}
	return out
}

func TestAddLegacyContribAliases(t *testing.T) {
	const (
		dbdotRef    = dbdotContribPrefix + "extension/awss3eventextension v0.0.1"
		contribRef  = legacyContribPrefix + "extension/pebbleextension v1.14.0"
		upstreamRef = "go.opentelemetry.io/collector/exporter/otlpexporter v0.160.0"
	)
	build := func() *protobufs.AvailableComponents {
		return &protobufs.AvailableComponents{
			Hash: []byte("base"),
			Components: map[string]*protobufs.ComponentDetails{
				"extensions": {SubComponentMap: map[string]*protobufs.ComponentDetails{
					"s3event": {Metadata: []*protobufs.KeyValue{codeNamespace(dbdotRef)}},
					"pebble":  {Metadata: []*protobufs.KeyValue{codeNamespace(contribRef)}},
				}},
				"exporters": {SubComponentMap: map[string]*protobufs.ComponentDetails{
					"otlp": {Metadata: []*protobufs.KeyValue{codeNamespace(upstreamRef)}},
				}},
			},
		}
	}

	ac := build()
	addLegacyContribAliases(ac)

	ext := ac.Components["extensions"].SubComponentMap
	// dbdot component: legacy alias first, real ref second, alias uses the pinned version.
	require.Equal(t, []string{
		legacyContribPrefix + "extension/awss3eventextension " + legacyContribAliasVersion,
		dbdotRef,
	}, values(ext["s3event"].Metadata))
	for _, kv := range ext["s3event"].Metadata {
		require.Equal(t, "code.namespace", kv.Key)
	}
	// non-dbdot components are untouched
	require.Equal(t, []string{contribRef}, values(ext["pebble"].Metadata))
	require.Equal(t, []string{upstreamRef}, values(ac.Components["exporters"].SubComponentMap["otlp"].Metadata))
	// hash changes when an alias was added, and is deterministic
	require.NotEqual(t, []byte("base"), ac.Hash)
	again := build()
	addLegacyContribAliases(again)
	require.Equal(t, ac.Hash, again.Hash)

	// no dbdot components: report and hash pass through unchanged
	plain := build()
	delete(plain.Components["extensions"].SubComponentMap, "s3event")
	addLegacyContribAliases(plain)
	require.Equal(t, []byte("base"), plain.Hash)
	require.Equal(t, []string{contribRef}, values(plain.Components["extensions"].SubComponentMap["pebble"].Metadata))
}
