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

package opampconnectionextension

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

func TestExtension_Start_RejectsSecondInstance(t *testing.T) {
	ext1 := newExtension(component.MustNewIDWithName("opamp_connection", "one"), zap.NewNop())
	ext2 := newExtension(component.MustNewIDWithName("opamp_connection", "two"), zap.NewNop())

	require.NoError(t, ext1.Start(context.Background(), nil))
	t.Cleanup(func() { _ = ext1.Shutdown(context.Background()) })

	err := ext2.Start(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "only one opamp_connection extension")
}

func TestExtension_Shutdown_AllowsSubsequentStart(t *testing.T) {
	ext1 := newExtension(component.MustNewIDWithName("opamp_connection", "first"), zap.NewNop())
	ext2 := newExtension(component.MustNewIDWithName("opamp_connection", "second"), zap.NewNop())

	require.NoError(t, ext1.Start(context.Background(), nil))
	require.NoError(t, ext1.Shutdown(context.Background()))

	// After the first extension shuts down, a second one can start in its
	// place (e.g. across a collector restart).
	require.NoError(t, ext2.Start(context.Background(), nil))
	t.Cleanup(func() { _ = ext2.Shutdown(context.Background()) })
}

func TestGetRegistry(t *testing.T) {
	id := component.MustNewIDWithName("opamp_connection", "lookup")
	ext := newExtension(id, zap.NewNop())

	require.Nil(t, GetRegistry(id), "should return nil before Start")

	require.NoError(t, ext.Start(context.Background(), nil))
	t.Cleanup(func() { _ = ext.Shutdown(context.Background()) })

	got := GetRegistry(id)
	require.NotNil(t, got)
	require.Equal(t, Registry(ext), got)

	require.Nil(t, GetRegistry(component.MustNewIDWithName("opamp_connection", "other")))
}

func TestForEachInstance(t *testing.T) {
	id := component.MustNewIDWithName("opamp_connection", "foreach")
	ext := newExtension(id, zap.NewNop())

	require.NoError(t, ext.Start(context.Background(), nil))
	t.Cleanup(func() { _ = ext.Shutdown(context.Background()) })

	var seen []Registry
	ForEachInstance(func(r Registry) {
		seen = append(seen, r)
	})

	require.Len(t, seen, 1)
	require.Equal(t, Registry(ext), seen[0])
}
