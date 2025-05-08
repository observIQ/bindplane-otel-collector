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

package grpcq_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/exporter/exporterhelper"

	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/internal/grpcq"
)

func TestSettings(t *testing.T) {
	settings := grpcq.Settings()

	assert.IsType(t, &grpcq.Encoding{}, settings.Encoding)

	assert.Len(t, settings.Sizers, 1)
	sizer, ok := settings.Sizers[exporterhelper.RequestSizerTypeBytes]
	assert.True(t, ok)
	assert.IsType(t, &grpcq.ByteSizer{}, sizer)
}
