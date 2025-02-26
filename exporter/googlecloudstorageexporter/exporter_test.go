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

package googlecloudstorageexporter // import "github.com/observiq/bindplane-otel-collector/exporter/googlecloudstorageexporter"

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/observiq/bindplane-otel-collector/exporter/googlecloudstorageexporter/internal/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

func Test_exporter_Capabilities(t *testing.T) {
	exp := &googleCloudStorageExporter{}
	capabilities := exp.Capabilities()
	require.False(t, capabilities.MutatesData)
}

func Test_exporter_metricsDataPusher(t *testing.T) {
	cfg := &Config{
		BucketName: "bucket",
		ProjectID:  "project",
		FolderName: "folder",
		ObjectPrefix: "prefix",
		Partition:  minutePartition,
	}

	testCases := []struct {
		desc        string
		mockGen     func(t *testing.T, input pmetric.Metrics, expectBuff []byte) (storageClient, marshaler)
		expectedErr error
	}{
		{
			desc: "marshal error",
			mockGen: func(t *testing.T, input pmetric.Metrics, _ []byte) (storageClient, marshaler) {
				mockStorageClient := mocks.NewMockStorageClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalMetrics(input).Return(nil, errors.New("marshal"))

				return mockStorageClient, mockMarshaler
			},
			expectedErr: errors.New("marshal"),
		},
		{
			desc: "Storage client error",
			mockGen: func(t *testing.T, input pmetric.Metrics, expectBuff []byte) (storageClient, marshaler) {
				mockStorageClient := mocks.NewMockStorageClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalMetrics(input).Return(expectBuff, nil)
				mockMarshaler.EXPECT().Format().Return("json")

				mockStorageClient.EXPECT().UploadObject(mock.Anything, mock.Anything, expectBuff).Return(errors.New("client"))

				return mockStorageClient, mockMarshaler
			},
			expectedErr: errors.New("client"),
		},
		{
			desc: "Successful push",
			mockGen: func(t *testing.T, input pmetric.Metrics, expectBuff []byte) (storageClient, marshaler) {
				mockStorageClient := mocks.NewMockStorageClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalMetrics(input).Return(expectBuff, nil)
				mockMarshaler.EXPECT().Format().Return("json")

				mockStorageClient.EXPECT().UploadObject(mock.Anything, mock.Anything, expectBuff).Return(nil)

				return mockStorageClient, mockMarshaler
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			md, expectedBytes := generateTestMetrics(t)
			mockStorageClient, mockMarshaler := tc.mockGen(t, md, expectedBytes)
			exp := &googleCloudStorageExporter{
				cfg:        cfg,
				storageClient: mockStorageClient,
				logger:     zap.NewNop(),
				marshaler:  mockMarshaler,
			}

			err := exp.metricsDataPusher(context.Background(), md)
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}

		})
	}
}

func Test_exporter_logsDataPusher(t *testing.T) {
	cfg := &Config{
		BucketName: "bucket",
		ProjectID:  "project",
		FolderName: "folder",
		ObjectPrefix: "prefix",
		Partition:  minutePartition,
	}

	testCases := []struct {
		desc        string
		mockGen     func(t *testing.T, input plog.Logs, expectBuff []byte) (storageClient, marshaler)
		expectedErr error
	}{
		{
			desc: "marshal error",
			mockGen: func(t *testing.T, input plog.Logs, _ []byte) (storageClient, marshaler) {
				mockStorageClient := mocks.NewMockStorageClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalLogs(input).Return(nil, errors.New("marshal"))

				return mockStorageClient, mockMarshaler
			},
			expectedErr: errors.New("marshal"),
		},
		{
			desc: "Storage client error",
			mockGen: func(t *testing.T, input plog.Logs, expectBuff []byte) (storageClient, marshaler) {
				mockStorageClient := mocks.NewMockStorageClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalLogs(input).Return(expectBuff, nil)
				mockMarshaler.EXPECT().Format().Return("json")

				mockStorageClient.EXPECT().UploadObject(mock.Anything, mock.Anything, expectBuff).Return(errors.New("client"))

				return mockStorageClient, mockMarshaler
			},
			expectedErr: errors.New("client"),
		},
		{
			desc: "Successful push",
			mockGen: func(t *testing.T, input plog.Logs, expectBuff []byte) (storageClient, marshaler) {
				mockStorageClient := mocks.NewMockStorageClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalLogs(input).Return(expectBuff, nil)
				mockMarshaler.EXPECT().Format().Return("json")

				mockStorageClient.EXPECT().UploadObject(mock.Anything, mock.Anything, expectBuff).Return(nil)

				return mockStorageClient, mockMarshaler
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ld, expectedBytes := generateTestLogs(t)
			mockStorageClient, mockMarshaler := tc.mockGen(t, ld, expectedBytes)
			exp := &googleCloudStorageExporter{
				cfg:        cfg,
				storageClient: mockStorageClient,
				logger:     zap.NewNop(),
				marshaler:  mockMarshaler,
			}

			err := exp.logsDataPusher(context.Background(), ld)
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}

		})
	}
}

func Test_exporter_traceDataPusher(t *testing.T) {
	cfg := &Config{
		BucketName: "bucket",
		ProjectID:  "project",
		FolderName: "folder",
		ObjectPrefix: "prefix",
		Partition:  minutePartition,
	}

	testCases := []struct {
		desc        string
		mockGen     func(t *testing.T, input ptrace.Traces, expectBuff []byte) (storageClient, marshaler)
		expectedErr error
	}{
		{
			desc: "marshal error",
			mockGen: func(t *testing.T, input ptrace.Traces, _ []byte) (storageClient, marshaler) {
				mockStorageClient := mocks.NewMockStorageClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalTraces(input).Return(nil, errors.New("marshal"))

				return mockStorageClient, mockMarshaler
			},
			expectedErr: errors.New("marshal"),
		},
		{
			desc: "Storage client error",
			mockGen: func(t *testing.T, input ptrace.Traces, expectBuff []byte) (storageClient, marshaler) {
				mockStorageClient := mocks.NewMockStorageClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalTraces(input).Return(expectBuff, nil)
				mockMarshaler.EXPECT().Format().Return("json")

				mockStorageClient.EXPECT().UploadObject(mock.Anything, mock.Anything, expectBuff).Return(errors.New("client"))

				return mockStorageClient, mockMarshaler
			},
			expectedErr: errors.New("client"),
		},
		{
			desc: "Successful push",
			mockGen: func(t *testing.T, input ptrace.Traces, expectBuff []byte) (storageClient, marshaler) {
				mockStorageClient := mocks.NewMockStorageClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalTraces(input).Return(expectBuff, nil)
				mockMarshaler.EXPECT().Format().Return("json")

				mockStorageClient.EXPECT().UploadObject(mock.Anything, mock.Anything, expectBuff).Return(nil)

				return mockStorageClient, mockMarshaler
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			td, expectedBytes := generateTestTraces(t)
			mockStorageClient, mockMarshaler := tc.mockGen(t, td, expectedBytes)
			exp := &googleCloudStorageExporter{
				cfg:        cfg,
				storageClient: mockStorageClient,
				logger:     zap.NewNop(),
				marshaler:  mockMarshaler,
			}

			err := exp.tracesDataPusher(context.Background(), td)
			if tc.expectedErr == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.expectedErr.Error())
			}

		})
	}
}

func Test_exporter_getObjectName(t *testing.T) {
	testCases := []struct {
		desc          string
		cfg           *Config
		telemetryType string
		expectedRegex string
	}{
		{
			desc: "Base Empty config",
			cfg: &Config{
				BucketName: "bucket",
				ProjectID:  "project",
				Partition:  minutePartition,
			},
			telemetryType: "metrics",
			expectedRegex: `^year=\d{4}/month=\d{2}/day=\d{2}/hour=\d{2}/minute=\d{2}/metrics_\d+\.json$`,
		},
		{
			desc: "Base Empty config hour",
			cfg: &Config{
				BucketName: "bucket",
				ProjectID:  "project",
				Partition:  hourPartition,
			},
			telemetryType: "metrics",
			expectedRegex: `^year=\d{4}/month=\d{2}/day=\d{2}/hour=\d{2}/metrics_\d+\.json$`,
		},
		{
			desc: "Full config",
			cfg: &Config{
				BucketName: "bucket",
				ProjectID:  "project",
				FolderName: "folder",
				ObjectPrefix: "prefix",
				Partition:  minutePartition,
			},
			telemetryType: "metrics",
			expectedRegex: `^folder/year=\d{4}/month=\d{2}/day=\d{2}/hour=\d{2}/minute=\d{2}/prefixmetrics_\d+\.json$`,
		},
	}

	for _, tc := range testCases {
		currentTc := tc
		t.Run(currentTc.desc, func(t *testing.T) {
			t.Parallel()
			mockMarshaller := mocks.NewMockMarshaler(t)
			mockMarshaller.EXPECT().Format().Return("json")

			exp := googleCloudStorageExporter{
				cfg:       currentTc.cfg,
				marshaler: mockMarshaller,
			}

			actual := exp.getObjectName(currentTc.telemetryType)
			require.Regexp(t, regexp.MustCompile(currentTc.expectedRegex), actual)
		})
	}
}
