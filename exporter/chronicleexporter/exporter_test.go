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

package chronicleexporter

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/observiq/bindplane-agent/exporter/chronicleexporter/internal/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLogsDataPusher(t *testing.T) {
	// Set up configuration, logger, and context
	cfg := Config{Region: "United States Multi-Region"}
	ctx := context.Background()

	testCases := []struct {
		desc          string
		setupExporter func() *chronicleExporter
		setupMocks    func(*chronicleExporter)
		expectedErr   string
	}{
		{
			desc: "successful push to Chronicle",
			setupExporter: func() *chronicleExporter {
				exporter := &chronicleExporter{
					endpoint:   baseEndpoint,
					cfg:        &cfg,
					logger:     zap.NewNop(),
					httpClient: http.DefaultClient,
				}
				httpmock.ActivateNonDefault(exporter.httpClient)
				return exporter
			},
			setupMocks: func(exporter *chronicleExporter) {
				httpmock.RegisterResponder("POST", baseEndpoint, httpmock.NewStringResponder(http.StatusOK, ""))

				marshaller := mocks.NewMockMarshaler(t)
				marshaller.On("MarshalRawLogs", mock.Anything, mock.Anything).Return([]byte("mock data"), nil)
				exporter.marshaler = marshaller
			},
			expectedErr: "",
		},
		{
			desc: "send request to Chronicle",
			setupExporter: func() *chronicleExporter {
				exporter := &chronicleExporter{
					endpoint:   baseEndpoint,
					cfg:        &cfg,
					logger:     zap.NewNop(),
					httpClient: http.DefaultClient,
				}
				httpmock.ActivateNonDefault(exporter.httpClient)
				return exporter
			},
			setupMocks: func(exporter *chronicleExporter) {
				// Register a responder that returns an error to simulate sending request failure
				httpmock.RegisterResponder("POST", baseEndpoint, httpmock.NewErrorResponder(errors.New("network error")))
				marshaller := mocks.NewMockMarshaler(t)
				marshaller.On("MarshalRawLogs", mock.Anything, mock.Anything).Return([]byte("mock data"), nil)
				exporter.marshaler = marshaller
			},
			expectedErr: "send request to Chronicle",
		},
		{
			desc: "marshaling logs fails",
			setupExporter: func() *chronicleExporter {
				exporter := &chronicleExporter{
					endpoint:   baseEndpoint,
					cfg:        &cfg,
					logger:     zap.NewNop(),
					httpClient: http.DefaultClient,
				}
				httpmock.ActivateNonDefault(exporter.httpClient)
				return exporter
			},
			setupMocks: func(exporter *chronicleExporter) {
				marshaller := mocks.NewMockMarshaler(t)
				marshaller.On("MarshalRawLogs", mock.Anything, mock.Anything).Return(nil, errors.New("marshaling error"))
				exporter.marshaler = marshaller
			},
			expectedErr: "marshal logs",
		},
		{
			desc: "received non-OK response from Chronicle",
			setupExporter: func() *chronicleExporter {
				exporter := &chronicleExporter{
					endpoint:   baseEndpoint,
					cfg:        &cfg,
					logger:     zap.NewNop(),
					httpClient: http.DefaultClient,
				}
				httpmock.ActivateNonDefault(exporter.httpClient)
				return exporter
			},
			setupMocks: func(exporter *chronicleExporter) {
				// Mock a non-OK HTTP response
				httpmock.RegisterResponder("POST", baseEndpoint, httpmock.NewStringResponder(http.StatusInternalServerError, "Internal Server Error"))

				marshaller := mocks.NewMockMarshaler(t)
				marshaller.On("MarshalRawLogs", mock.Anything, mock.Anything).Return([]byte("mock data"), nil)
				exporter.marshaler = marshaller
			},
			expectedErr: "received non-OK response from Chronicle: 500",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			exporter := tc.setupExporter()
			defer httpmock.DeactivateAndReset()

			tc.setupMocks(exporter)

			// Create a dummy plog.Logs to pass to logsDataPusher
			logs := mockLogs(mockLogRecord(t, "Test body", map[string]any{"key1": "value1"}))

			err := exporter.logsDataPusher(ctx, logs)

			if tc.expectedErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedErr)
			}

			// Verify the expected number of calls were made
			if tc.expectedErr == "" {
				info := httpmock.GetCallCountInfo()
				expectedMethod := "POST " + baseEndpoint
				require.Equal(t, 1, info[expectedMethod], "Expected number of calls to %s is not met", expectedMethod)
			}
		})
	}
}
func TestBuildEndpoint(t *testing.T) {
	testCases := []struct {
		name             string
		cfg              Config
		expectedEndpoint string
	}{
		{
			name:             "Europe Multi-Region",
			cfg:              Config{Region: "Europe Multi-Region"},
			expectedEndpoint: "https://europe-backstory.googleapis.com/v2/unstructuredlogentries:batchCreate",
		},
		{
			name:             "Frankfurt",
			cfg:              Config{Region: "Frankfurt"},
			expectedEndpoint: "https://europe-west3-backstory.googleapis.com/v2/unstructuredlogentries:batchCreate",
		},
		{
			name:             "London",
			cfg:              Config{Region: "London"},
			expectedEndpoint: "http://europe-west2-backstory.googleapis.com/v2/unstructuredlogentries:batchCreate",
		},
		{
			name:             "Singapore",
			cfg:              Config{Region: "Singapore"},
			expectedEndpoint: "https://asia-southeast1-backstory.googleapis.com/v2/unstructuredlogentries:batchCreate",
		},
		{
			name:             "default case (no region)",
			cfg:              Config{},
			expectedEndpoint: "https://malachiteingestion-pa.googleapis.com/v2/unstructuredlogentries:batchCreate",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := buildEndpoint(&tc.cfg)
			require.Equal(t, tc.expectedEndpoint, endpoint, "Endpoint does not match for test case: %s", tc.name)
		})
	}
}

func Test_exporter_Capabilities(t *testing.T) {
	exp := &chronicleExporter{}
	capabilities := exp.Capabilities()
	require.False(t, capabilities.MutatesData)
}
