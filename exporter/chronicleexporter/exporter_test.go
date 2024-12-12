// Copyright observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package chronicleexporter

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api"
	"github.com/observiq/bindplane-otel-collector/exporter/chronicleexporter/protos/api/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumererror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLogsDataPusher(t *testing.T) {

	// Set up configuration, logger, and context
	cfg := Config{Endpoint: defaultEndpoint}
	ctx := context.Background()

	testCases := []struct {
		desc          string
		setupExporter func() *chronicleExporter
		setupMocks    func(*mocks.MockIngestionServiceV2Client)
		expectedErr   string
		permanentErr  bool
	}{
		{
			desc: "successful push to Chronicle",
			setupExporter: func() *chronicleExporter {
				mockClient := mocks.NewMockIngestionServiceV2Client(gomock.NewController(t))
				marshaller := NewMockMarshaler(t)
				marshaller.On("MarshalRawLogs", mock.Anything, mock.Anything).Return([]*api.BatchCreateLogsRequest{{}}, nil)
				return &chronicleExporter{
					cfg:        &cfg,
					metrics:    newHostMetricsReporter([]byte{}, []byte{}, "", cfg.Namespace),
					set:        componenttest.NewNopTelemetrySettings(),
					grpcClient: mockClient,
					marshaler:  marshaller,
				}
			},
			setupMocks: func(mockClient *mocks.MockIngestionServiceV2Client) {
				mockClient.EXPECT().BatchCreateLogs(gomock.Any(), gomock.Any(), gomock.Any()).Return(&api.BatchCreateLogsResponse{}, nil)
			},
			expectedErr: "",
		},
		{
			desc: "upload to Chronicle fails (transient)",
			setupExporter: func() *chronicleExporter {
				mockClient := mocks.NewMockIngestionServiceV2Client(gomock.NewController(t))
				marshaller := NewMockMarshaler(t)
				marshaller.On("MarshalRawLogs", mock.Anything, mock.Anything).Return([]*api.BatchCreateLogsRequest{{}}, nil)
				return &chronicleExporter{
					cfg:        &cfg,
					metrics:    newHostMetricsReporter([]byte{}, []byte{}, "", cfg.Namespace),
					set:        componenttest.NewNopTelemetrySettings(),
					grpcClient: mockClient,
					marshaler:  marshaller,
				}
			},
			setupMocks: func(mockClient *mocks.MockIngestionServiceV2Client) {
				// Simulate an error returned from the Chronicle service
				err := status.Error(codes.Unavailable, "service unavailable")
				mockClient.EXPECT().BatchCreateLogs(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err)
			},
			expectedErr: "service unavailable",
		},
		{
			desc: "upload to Chronicle fails (permanent)",
			setupExporter: func() *chronicleExporter {
				mockClient := mocks.NewMockIngestionServiceV2Client(gomock.NewController(t))
				marshaller := NewMockMarshaler(t)
				marshaller.On("MarshalRawLogs", mock.Anything, mock.Anything).Return([]*api.BatchCreateLogsRequest{{}}, nil)
				return &chronicleExporter{
					cfg:        &cfg,
					metrics:    newHostMetricsReporter([]byte{}, []byte{}, "", cfg.Namespace),
					set:        componenttest.NewNopTelemetrySettings(),
					grpcClient: mockClient,
					marshaler:  marshaller,
				}
			},
			setupMocks: func(mockClient *mocks.MockIngestionServiceV2Client) {
				err := status.Error(codes.InvalidArgument, "Invalid argument detected.")
				mockClient.EXPECT().BatchCreateLogs(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err)
			},
			expectedErr:  "Invalid argument detected.",
			permanentErr: true,
		},
		{
			desc: "marshaler error",
			setupExporter: func() *chronicleExporter {
				mockClient := mocks.NewMockIngestionServiceV2Client(gomock.NewController(t))
				marshaller := NewMockMarshaler(t)
				// Simulate an error during log marshaling
				marshaller.On("MarshalRawLogs", mock.Anything, mock.Anything).Return(nil, errors.New("marshal error"))
				return &chronicleExporter{
					cfg:        &cfg,
					metrics:    newHostMetricsReporter([]byte{}, []byte{}, "", cfg.Namespace),
					set:        componenttest.NewNopTelemetrySettings(),
					grpcClient: mockClient,
					marshaler:  marshaller,
				}
			},
			setupMocks: func(_ *mocks.MockIngestionServiceV2Client) {
				// No need to setup mocks for the client as the error occurs before the client is used
			},
			expectedErr: "marshal error",
		},
		{
			desc: "empty log records",
			setupExporter: func() *chronicleExporter {
				mockClient := mocks.NewMockIngestionServiceV2Client(gomock.NewController(t))
				marshaller := NewMockMarshaler(t)
				// Return an empty slice to simulate no logs to push
				marshaller.On("MarshalRawLogs", mock.Anything, mock.Anything).Return([]*api.BatchCreateLogsRequest{}, nil)
				return &chronicleExporter{
					cfg:        &cfg,
					metrics:    newHostMetricsReporter([]byte{}, []byte{}, "", cfg.Namespace),
					set:        componenttest.NewNopTelemetrySettings(),
					grpcClient: mockClient,
					marshaler:  marshaller,
				}
			},
			setupMocks: func(_ *mocks.MockIngestionServiceV2Client) {
				// Expect no calls to BatchCreateLogs since there are no logs to push
			},
			expectedErr: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			exporter := tc.setupExporter()
			tc.setupMocks(exporter.grpcClient.(*mocks.MockIngestionServiceV2Client))

			// Create a dummy plog.Logs to pass to logsDataPusher
			logs := mockLogs(mockLogRecord("Test body", map[string]any{"key1": "value1"}))

			err := exporter.logsDataPusher(ctx, logs)

			if tc.expectedErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr)
				if tc.permanentErr {
					require.True(t, consumererror.IsPermanent(err), "Expected error to be permanent")
				} else {
					require.False(t, consumererror.IsPermanent(err), "Expected error to be transient")
				}
			}
		})
	}
}
