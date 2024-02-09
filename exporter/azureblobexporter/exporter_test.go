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

package azureblobexporter // import "github.com/observiq/bindplane-agent/exporter/azureblobexporter"

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/observiq/bindplane-agent/exporter/azureblobexporter/internal/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

func Test_exporter_Capabilities(t *testing.T) {
	exp := &azureBlobExporter{}
	capabilities := exp.Capabilities()
	require.False(t, capabilities.MutatesData)
}

func Test_exporter_metricsDataPusher(t *testing.T) {
	cfg := &Config{
		Container:  "container",
		BlobPrefix: "blob",
		RootFolder: "root",
		Partition:  minutePartition,
	}

	testCases := []struct {
		desc        string
		mockGen     func(t *testing.T, input pmetric.Metrics, expectBuff []byte) (blobClient, marshaler)
		expectedErr error
	}{
		{
			desc: "marshal error",
			mockGen: func(t *testing.T, input pmetric.Metrics, _ []byte) (blobClient, marshaler) {
				mockBlobClient := mocks.NewMockBlobClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalMetrics(input).Return(nil, errors.New("marshal"))

				return mockBlobClient, mockMarshaler
			},
			expectedErr: errors.New("marshal"),
		},
		{
			desc: "Blob client error",
			mockGen: func(t *testing.T, input pmetric.Metrics, expectBuff []byte) (blobClient, marshaler) {
				mockBlobClient := mocks.NewMockBlobClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalMetrics(input).Return(expectBuff, nil)
				mockMarshaler.EXPECT().Format().Return("json")

				mockBlobClient.EXPECT().UploadBuffer(mock.Anything, cfg.Container, mock.Anything, expectBuff).Return(errors.New("client"))

				return mockBlobClient, mockMarshaler
			},
			expectedErr: errors.New("client"),
		},
		{
			desc: "Successful push",
			mockGen: func(t *testing.T, input pmetric.Metrics, expectBuff []byte) (blobClient, marshaler) {
				mockBlobClient := mocks.NewMockBlobClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalMetrics(input).Return(expectBuff, nil)
				mockMarshaler.EXPECT().Format().Return("json")

				mockBlobClient.EXPECT().UploadBuffer(mock.Anything, cfg.Container, mock.Anything, expectBuff).Return(nil)

				return mockBlobClient, mockMarshaler
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			md, expectedBytes := generateTestMetrics(t)
			mockBlobClient, mockMarshaler := tc.mockGen(t, md, expectedBytes)
			exp := &azureBlobExporter{
				cfg:        cfg,
				blobClient: mockBlobClient,
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
		Container:  "container",
		BlobPrefix: "blob",
		RootFolder: "root",
		Partition:  minutePartition,
	}

	testCases := []struct {
		desc        string
		mockGen     func(t *testing.T, input plog.Logs, expectBuff []byte) (blobClient, marshaler)
		expectedErr error
	}{
		{
			desc: "marshal error",
			mockGen: func(t *testing.T, input plog.Logs, _ []byte) (blobClient, marshaler) {
				mockBlobClient := mocks.NewMockBlobClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalLogs(input).Return(nil, errors.New("marshal"))

				return mockBlobClient, mockMarshaler
			},
			expectedErr: errors.New("marshal"),
		},
		{
			desc: "Blob client error",
			mockGen: func(t *testing.T, input plog.Logs, expectBuff []byte) (blobClient, marshaler) {
				mockBlobClient := mocks.NewMockBlobClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalLogs(input).Return(expectBuff, nil)
				mockMarshaler.EXPECT().Format().Return("json")

				mockBlobClient.EXPECT().UploadBuffer(mock.Anything, cfg.Container, mock.Anything, expectBuff).Return(errors.New("client"))

				return mockBlobClient, mockMarshaler
			},
			expectedErr: errors.New("client"),
		},
		{
			desc: "Successful push",
			mockGen: func(t *testing.T, input plog.Logs, expectBuff []byte) (blobClient, marshaler) {
				mockBlobClient := mocks.NewMockBlobClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalLogs(input).Return(expectBuff, nil)
				mockMarshaler.EXPECT().Format().Return("json")

				mockBlobClient.EXPECT().UploadBuffer(mock.Anything, cfg.Container, mock.Anything, expectBuff).Return(nil)

				return mockBlobClient, mockMarshaler
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ld, expectedBytes := generateTestLogs(t)
			mockBlobClient, mockMarshaler := tc.mockGen(t, ld, expectedBytes)
			exp := &azureBlobExporter{
				cfg:        cfg,
				blobClient: mockBlobClient,
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
		Container:  "container",
		BlobPrefix: "blob",
		RootFolder: "root",
		Partition:  minutePartition,
	}

	testCases := []struct {
		desc        string
		mockGen     func(t *testing.T, input ptrace.Traces, expectBuff []byte) (blobClient, marshaler)
		expectedErr error
	}{
		{
			desc: "marshal error",
			mockGen: func(t *testing.T, input ptrace.Traces, _ []byte) (blobClient, marshaler) {
				mockBlobClient := mocks.NewMockBlobClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalTraces(input).Return(nil, errors.New("marshal"))

				return mockBlobClient, mockMarshaler
			},
			expectedErr: errors.New("marshal"),
		},
		{
			desc: "Blob client error",
			mockGen: func(t *testing.T, input ptrace.Traces, expectBuff []byte) (blobClient, marshaler) {
				mockBlobClient := mocks.NewMockBlobClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalTraces(input).Return(expectBuff, nil)
				mockMarshaler.EXPECT().Format().Return("json")

				mockBlobClient.EXPECT().UploadBuffer(mock.Anything, cfg.Container, mock.Anything, expectBuff).Return(errors.New("client"))

				return mockBlobClient, mockMarshaler
			},
			expectedErr: errors.New("client"),
		},
		{
			desc: "Successful push",
			mockGen: func(t *testing.T, input ptrace.Traces, expectBuff []byte) (blobClient, marshaler) {
				mockBlobClient := mocks.NewMockBlobClient(t)
				mockMarshaler := mocks.NewMockMarshaler(t)

				mockMarshaler.EXPECT().MarshalTraces(input).Return(expectBuff, nil)
				mockMarshaler.EXPECT().Format().Return("json")

				mockBlobClient.EXPECT().UploadBuffer(mock.Anything, cfg.Container, mock.Anything, expectBuff).Return(nil)

				return mockBlobClient, mockMarshaler
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			td, expectedBytes := generateTestTraces(t)
			mockBlobClient, mockMarshaler := tc.mockGen(t, td, expectedBytes)
			exp := &azureBlobExporter{
				cfg:        cfg,
				blobClient: mockBlobClient,
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

func Test_exporter_getBlobName(t *testing.T) {
	testCases := []struct {
		desc          string
		cfg           *Config
		telemetryType string
		expectedRegex string
	}{
		{
			desc: "Base Empty config",
			cfg: &Config{
				Container: "otel",
				Partition: minutePartition,
			},
			telemetryType: "metrics",
			expectedRegex: `^year=\d{4}/month=\d{2}/day=\d{2}/hour=\d{2}/minute=\d{2}/metrics_\d+\.json$`,
		},
		{
			desc: "Base Empty config hour",
			cfg: &Config{
				Container: "otel",
				Partition: hourPartition,
			},
			telemetryType: "metrics",
			expectedRegex: `^year=\d{4}/month=\d{2}/day=\d{2}/hour=\d{2}/metrics_\d+\.json$`,
		},
		{
			desc: "Full config",
			cfg: &Config{
				Container:  "otel",
				RootFolder: "root",
				BlobPrefix: "blob",
				Partition:  minutePartition,
			},
			telemetryType: "metrics",
			expectedRegex: `^root/year=\d{4}/month=\d{2}/day=\d{2}/hour=\d{2}/minute=\d{2}/blobmetrics_\d+\.json$`,
		},
	}

	for _, tc := range testCases {
		currentTc := tc
		t.Run(currentTc.desc, func(t *testing.T) {
			t.Parallel()
			mockMarshaller := mocks.NewMockMarshaler(t)
			mockMarshaller.EXPECT().Format().Return("json")

			exp := azureBlobExporter{
				cfg:       currentTc.cfg,
				marshaler: mockMarshaller,
			}

			actual := exp.getBlobName(currentTc.telemetryType)
			require.Regexp(t, regexp.MustCompile(currentTc.expectedRegex), actual)
		})
	}
}
