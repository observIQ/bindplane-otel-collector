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

package service

import (
	"context"
	"errors"
	"testing"

	colmocks "github.com/observiq/bindplane-otel-collector/collector/mocks"
	"github.com/observiq/bindplane-otel-collector/opamp/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestNewManagedCollectorService_BadManagerConfig tests NewManagedCollectorService
// for a bad manger config. This function starts an entire observiq client which is tested
// in it's own package so we don't do robust testing here.
func TestNewManagedCollectorService_BadManagerConfig(t *testing.T) {
	mockCol := colmocks.NewMockCollector(t)
	managedService, err := NewManagedCollectorService(mockCol, zap.NewNop(), "./bad_manger.yaml", "./bad_collector.yaml", "./bad_logging.yaml")
	assert.ErrorContains(t, err, "failed to parse manager config")
	assert.Nil(t, managedService)
}

func TestManageCollectorServiceStart(t *testing.T) {
	testCases := []struct {
		desc          string
		setupMocks    func(*mocks.MockClient)
		expectedError error
	}{
		{
			desc: "Client fails to connect",
			setupMocks: func(m *mocks.MockClient) {
				m.On("Connect", mock.Anything).Return(errors.New("oops"))
			},
			expectedError: errors.New("oops"),
		},
		{
			desc: "Client successfully connects",
			setupMocks: func(m *mocks.MockClient) {
				m.On("Connect", mock.Anything).Return(nil)
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			mockClient := mocks.NewMockClient(t)

			tc.setupMocks(mockClient)

			m := &ManagedCollectorService{
				client: mockClient,
				logger: zap.NewNop(),
			}

			err := m.Start(context.Background())
			if tc.expectedError == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.expectedError.Error())
			}
		})
	}
}

func TestManageCollectorServiceStop(t *testing.T) {
	testCases := []struct {
		desc          string
		setupMocks    func(*mocks.MockClient)
		expectedError error
	}{
		{
			desc: "Client fails to disconnect",
			setupMocks: func(m *mocks.MockClient) {
				m.On("Disconnect", mock.Anything).Return(errors.New("oops"))
			},
			expectedError: errors.New("oops"),
		},
		{
			desc: "Client successfully disconnects",
			setupMocks: func(m *mocks.MockClient) {
				m.On("Disconnect", mock.Anything).Return(nil)
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			mockClient := mocks.NewMockClient(t)

			tc.setupMocks(mockClient)

			m := &ManagedCollectorService{
				client: mockClient,
				logger: zap.NewNop(),
			}

			err := m.Stop(context.Background())
			if tc.expectedError == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.expectedError.Error())
			}
		})
	}
}

func TestManageCollectorServiceError(t *testing.T) {
	// Just test we return a non-nil channel
	m := &ManagedCollectorService{}
	errChan := m.Error()
	require.NotNil(t, errChan)
}
