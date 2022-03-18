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

package cloudwatch

import (
	"context"
	"testing"

	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/open-telemetry/opentelemetry-log-collection/testutil"
	"github.com/stretchr/testify/require"
)

func TestPersisterCache(t *testing.T) {
	ctx := context.TODO()
	stubDatabase := testutil.NewMockPersister("stub")
	p := persister{
		DB: operator.NewScopedPersister("test", stubDatabase),
	}
	err := p.Write(ctx, "key", int64(1620666055012))
	require.NoError(t, err)

	value, readErr := p.Read(ctx, "key")
	require.NoError(t, readErr)
	require.Equal(t, int64(1620666055012), value)
}

func TestPersisterLoad(t *testing.T) {
	ctx := context.TODO()
	p := testutil.NewMockPersister("mock")
	cwP := persister{
		DB: p,
	}
	err := cwP.Write(ctx, "key", 1620666055012)
	require.NoError(t, err)

	value, syncErr := cwP.Read(ctx, "key")
	require.NoError(t, syncErr)
	require.Equal(t, int64(1620666055012), value)
}

func TestPersistentLoadNoKey(t *testing.T) {
	ctx := context.TODO()

	p := testutil.NewMockPersister("mock")
	cwP := persister{
		DB: p,
	}
	value, err := cwP.Read(ctx, "key")
	require.NoError(t, err)
	require.Equal(t, int64(0), value)
}
