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

package opamp

import (
	"testing"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/require"
)

func TestStringKeyValue(t *testing.T) {
	key, value := "key", "value"
	expected := &protobufs.KeyValue{
		Key: key,
		Value: &protobufs.AnyValue{
			Value: &protobufs.AnyValue_StringValue{StringValue: value},
		},
	}

	actual := StringKeyValue(key, value)
	require.Equal(t, expected, actual)
}

func TestComputeHash(t *testing.T) {
	expected := []byte{0xc2, 0xae, 0xcc, 0xc4, 0x2d, 0x2a, 0x57, 0x9c, 0x28, 0x1d, 0xaa, 0xe7, 0xe4, 0x64, 0xa1, 0x4d, 0x74, 0x79, 0x24, 0x15, 0x9e, 0x28, 0x61, 0x7a, 0xd0, 0x18, 0x50, 0xf0, 0xdd, 0x1b, 0xd1, 0x35}
	actual := ComputeHash([]byte("hellow world"))
	require.Equal(t, expected, actual)
}
