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

package action

import (
	"testing"

	"github.com/observiq/bindplane-otel-collector/updater/internal/service/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestServiceStartAction(t *testing.T) {
	svc := mocks.NewMockService(t)
	ssa := NewServiceStartAction(svc)

	svc.On("Stop").Once().Return(nil)

	err := ssa.Rollback()
	require.NoError(t, err)
}

func TestServiceStopAction(t *testing.T) {
	svc := mocks.NewMockService(t)
	ssa := NewServiceStopAction(svc)

	svc.On("Start").Once().Return(nil)

	err := ssa.Rollback()
	require.NoError(t, err)
}

func TestServiceUpdateAction(t *testing.T) {
	svc := mocks.NewMockService(t)
	sua := NewServiceUpdateAction(zaptest.NewLogger(t), "./testdata")
	sua.backupSvc = svc

	svc.On("Update").Once().Return(nil)

	err := sua.Rollback()
	require.NoError(t, err)
}
