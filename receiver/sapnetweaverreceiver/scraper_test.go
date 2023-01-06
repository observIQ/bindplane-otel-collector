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

package sapnetweaverreceiver // import "github.com/observiq/observiq-otel-collector/receiver/sapnetweaverreceiver"

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/multierr"

	"github.com/observiq/observiq-otel-collector/receiver/sapnetweaverreceiver/internal/mocks"
	"github.com/observiq/observiq-otel-collector/receiver/sapnetweaverreceiver/internal/models"
)

func TestScraperStart(t *testing.T) {
	testcases := []struct {
		desc        string
		scraper     *sapNetweaverScraper
		expectError bool
	}{
		{
			desc: "Bad Config",
			scraper: &sapNetweaverScraper{
				cfg: &Config{
					HTTPClientSettings: confighttp.HTTPClientSettings{
						Endpoint: defaultEndpoint,
						TLSSetting: configtls.TLSClientSetting{
							TLSSetting: configtls.TLSSetting{
								CAFile: "/non/existent",
							},
						},
					},
				},
				settings: componenttest.NewNopTelemetrySettings(),
			},
			expectError: true,
		},
		{
			desc: "Valid Config",
			scraper: &sapNetweaverScraper{
				cfg: &Config{
					Username: "root",
					Password: "password",
					HTTPClientSettings: confighttp.HTTPClientSettings{
						TLSSetting: configtls.TLSClientSetting{},
						Endpoint:   defaultEndpoint,
					},
				},
				settings: componenttest.NewNopTelemetrySettings(),
			},
			expectError: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			mockService := mocks.MockWebService{}
			mockService.On("GetInstanceProperties").Return(models.GetInstancePropertiesResponse{XMLName: xml.Name{}}, nil)

			tc.scraper.service = &mockService
			err := tc.scraper.start(context.Background(), componenttest.NewNopHost())
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestScraperScrape(t *testing.T) {
	alertTreeResponseData := loadAPIResponseData(t, "api-responses", "alert-tree.xml")
	var alertTreeResponse *models.GetAlertTreeResponse
	err := xml.Unmarshal(alertTreeResponseData, &alertTreeResponse)
	require.NoError(t, err)

	enqGetLockTableResponseData := loadAPIResponseData(t, "api-responses", "lock-table.xml")
	var enqGetLockTableResponse *models.EnqGetLockTableResponse
	err = xml.Unmarshal(enqGetLockTableResponseData, &enqGetLockTableResponse)
	require.NoError(t, err)

	getCurrentInstanceResponseData := loadAPIResponseData(t, "api-responses", "current-instance.xml")
	var getCurrentInstanceResponse *models.GetInstancePropertiesResponse
	err = xml.Unmarshal(getCurrentInstanceResponseData, &getCurrentInstanceResponse)
	require.NoError(t, err)

	mockService := mocks.MockWebService{}
	mockService.On("EnqGetStatistic").Return(nil, nil)
	mockService.On("GetAlertTree").Return(alertTreeResponse, nil)
	mockService.On("EnqGetLockTable").Return(enqGetLockTableResponse, nil)
	mockService.On("GetInstanceProperties").Return(getCurrentInstanceResponse, nil)

	cfg := createDefaultConfig().(*Config)
	cfg.Endpoint = defaultEndpoint
	cfg.Username = "root"
	cfg.Password = "password"

	testClient, err := newSoapClient(cfg, componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	scraper := newSapNetweaverScraper(receivertest.NewNopCreateSettings(), createDefaultConfig().(*Config))
	scraper.service = &mockService
	scraper.client = testClient

	actualMetrics, err := scraper.scrape(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, actualMetrics.ResourceMetrics().Len())
	require.Equal(t, 16, actualMetrics.DataPointCount())
	require.Equal(t, 13, actualMetrics.MetricCount())

	require.EqualValues(t, "sap-app", scraper.hostname)
	require.EqualValues(t, "sap-inst", scraper.instance)

	for i := 0; i < actualMetrics.ResourceMetrics().Len(); i++ {
		ilm := actualMetrics.ResourceMetrics().At(i).ScopeMetrics()
		require.Equal(t, 1, ilm.Len())

		ms := ilm.At(0).Metrics()
		for i := 0; i < ms.Len(); i++ {
			m := ms.At(i)
			switch m.Name() {
			case "sapnetweaver.short_dumps.rate":
				dps := m.Sum().DataPoints()
				require.Equal(t, 1, dps.Len())
				require.Equal(t, int64(1), dps.At(0).IntValue())
			case "sapnetweaver.work_processes.active.count":
				dps := m.Sum().DataPoints()
				require.Equal(t, 1, dps.Len())
				require.Equal(t, int64(2), dps.At(0).IntValue())
			case "sapnetweaver.icm_availability":
				dps := m.Sum().DataPoints()
				require.Equal(t, 4, dps.Len())
				attributeMappings := map[string]int64{}
				for j := 0; j < dps.Len(); j++ {
					dp := dps.At(j)
					method := dp.Attributes().AsRaw()
					label := fmt.Sprintf("%s method:%s", m.Name(), method)
					attributeMappings[label] = dp.IntValue()
				}
				require.Equal(t, map[string]int64{
					"sapnetweaver.icm_availability method:map[state:green]":  int64(1),
					"sapnetweaver.icm_availability method:map[state:grey]":   int64(0),
					"sapnetweaver.icm_availability method:map[state:red]":    int64(0),
					"sapnetweaver.icm_availability method:map[state:yellow]": int64(0),
				},
					attributeMappings)
			case "sapnetweaver.host.spool_list.used":
				dps := m.Sum().DataPoints()
				require.Equal(t, 1, dps.Len())
				require.Equal(t, int64(3), dps.At(0).IntValue())
			case "sapnetweaver.host.memory.virtual.swap":
				dps := m.Gauge().DataPoints()
				require.Equal(t, 1, dps.Len())
				require.Equal(t, int64(4)*MBToBytes, dps.At(0).IntValue())
			case "sapnetweaver.host.cpu_utilization":
				dps := m.Gauge().DataPoints()
				require.Equal(t, 1, dps.Len())
				require.Equal(t, int64(5), dps.At(0).IntValue())
			case "sapnetweaver.host.memory.virtual.overhead":
				dps := m.Gauge().DataPoints()
				require.Equal(t, 1, dps.Len())
				require.Equal(t, int64(6)*MBToBytes, dps.At(0).IntValue())
			case "sapnetweaver.sessions.http.count":
				dps := m.Sum().DataPoints()
				require.Equal(t, 1, dps.Len())
				require.Equal(t, int64(7), dps.At(0).IntValue())
			case "sapnetweaver.sessions.security.count":
				dps := m.Sum().DataPoints()
				require.Equal(t, 1, dps.Len())
				require.Equal(t, int64(8), dps.At(0).IntValue())
			case "sapnetweaver.sessions.web.count":
				dps := m.Sum().DataPoints()
				require.Equal(t, 1, dps.Len())
				require.Equal(t, int64(9), dps.At(0).IntValue())
			case "sapnetweaver.sessions.browser.count":
				dps := m.Sum().DataPoints()
				require.Equal(t, 1, dps.Len())
				require.Equal(t, int64(10), dps.At(0).IntValue())
			case "sapnetweaver.sessions.ejb.count":
				dps := m.Sum().DataPoints()
				require.Equal(t, 1, dps.Len())
				require.Equal(t, int64(11), dps.At(0).IntValue())
			case "sapnetweaver.locks.enqueue.count":
				dps := m.Sum().DataPoints()
				require.Equal(t, 1, dps.Len())
				require.Equal(t, int64(3), dps.At(0).IntValue())
			}
		}
	}
}

func TestScraperScrapeHyphenResponse(t *testing.T) {
	alertTreeResponseData := loadAPIResponseData(t, "api-responses", "hyphen-alert-tree.xml")
	var alertTreeResponse *models.GetAlertTreeResponse
	err := xml.Unmarshal(alertTreeResponseData, &alertTreeResponse)
	require.NoError(t, err)

	enqGetLockTableResponseData := loadAPIResponseData(t, "api-responses", "empty-lock-table.xml")
	var enqGetLockTableResponse *models.EnqGetLockTableResponse
	err = xml.Unmarshal(enqGetLockTableResponseData, &enqGetLockTableResponse)
	require.NoError(t, err)

	getCurrentInstanceResponseData := loadAPIResponseData(t, "api-responses", "empty-current-instance.xml")
	var getCurrentInstanceResponse *models.GetInstancePropertiesResponse
	err = xml.Unmarshal(getCurrentInstanceResponseData, &getCurrentInstanceResponse)
	require.NoError(t, err)

	mockService := mocks.MockWebService{}
	mockService.On("EnqGetStatistic").Return(nil, nil)
	mockService.On("GetAlertTree").Return(alertTreeResponse, nil)
	mockService.On("EnqGetLockTable").Return(enqGetLockTableResponse, nil)
	mockService.On("GetInstanceProperties").Return(getCurrentInstanceResponse, nil)

	cfg := createDefaultConfig().(*Config)
	cfg.Endpoint = defaultEndpoint
	cfg.Username = "root"
	cfg.Password = "password"

	testClient, err := newSoapClient(cfg, componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	scraper := newSapNetweaverScraper(receivertest.NewNopCreateSettings(), createDefaultConfig().(*Config))
	scraper.service = &mockService
	scraper.client = testClient

	actualMetrics, err := scraper.scrape(context.Background())
	require.EqualError(t, multierr.Combine(
		errors.New("failed to collect metric CPU_Utilization: '-' value found"),
		errors.New("failed to collect metric Memory Overhead: '-' value found"),
		errors.New("failed to collect metric Memory Swapped Out: '-' value found"),
		errors.New("failed to collect metric CurrentHttpSessions: '-' value found"),
		errors.New("failed to collect metric CurrentSecuritySessions: '-' value found"),
		errors.New("failed to collect metric Total Number of Work Processes: '-' value found"),
		errors.New("failed to collect metric Web Sessions: '-' value found"),
		errors.New("failed to collect metric Browser Sessions: '-' value found"),
		errors.New("failed to collect metric EJB Sessions: '-' value found"),
		errors.New("failed to collect metric ICM: invalid control state color value"),
		errors.New("failed to collect metric HostspoolListUsed: '-' value found"),
		errors.New("failed to collect metric Shortdumps Frequency: '-' value found"),
	), err.Error())

	require.Error(t, err)
	require.Equal(t, 1, actualMetrics.ResourceMetrics().Len())
	require.Equal(t, 1, actualMetrics.DataPointCount())
	require.Equal(t, 1, actualMetrics.MetricCount())

	require.EqualValues(t, "", scraper.hostname)
	require.EqualValues(t, "", scraper.instance)

}

func TestScraperScrapeUnknownResponse(t *testing.T) {
	alertTreeResponseData := loadAPIResponseData(t, "api-responses", "unknown-value-alert-tree.xml")
	var alertTreeResponse *models.GetAlertTreeResponse
	err := xml.Unmarshal(alertTreeResponseData, &alertTreeResponse)
	require.NoError(t, err)

	enqGetLockTableResponseData := loadAPIResponseData(t, "api-responses", "empty-lock-table.xml")
	var enqGetLockTableResponse *models.EnqGetLockTableResponse
	err = xml.Unmarshal(enqGetLockTableResponseData, &enqGetLockTableResponse)
	require.NoError(t, err)

	getCurrentInstanceResponseData := loadAPIResponseData(t, "api-responses", "empty-current-instance.xml")
	var getCurrentInstanceResponse *models.GetInstancePropertiesResponse
	err = xml.Unmarshal(getCurrentInstanceResponseData, &getCurrentInstanceResponse)
	require.NoError(t, err)

	mockService := mocks.MockWebService{}
	mockService.On("EnqGetStatistic").Return(nil, nil)
	mockService.On("GetAlertTree").Return(alertTreeResponse, nil)
	mockService.On("EnqGetLockTable").Return(enqGetLockTableResponse, nil)
	mockService.On("GetInstanceProperties").Return(getCurrentInstanceResponse, nil)

	cfg := createDefaultConfig().(*Config)
	cfg.Endpoint = defaultEndpoint
	cfg.Username = "root"
	cfg.Password = "password"

	testClient, err := newSoapClient(cfg, componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	scraper := newSapNetweaverScraper(receivertest.NewNopCreateSettings(), createDefaultConfig().(*Config))
	scraper.service = &mockService
	scraper.client = testClient

	actualMetrics, err := scraper.scrape(context.Background())
	require.EqualError(t, multierr.Combine(
		errors.New("failed to parse int64 for SapnetweaverHostCPUUtilization, value was $: strconv.ParseInt: parsing \"$\": invalid syntax"),
		errors.New("failed to parse int64 for SapnetweaverHostMemoryVirtualOverhead, value was $: strconv.ParseInt: parsing \"$\": invalid syntax"),
		errors.New("failed to parse int64 for SapnetweaverHostMemoryVirtualSwap, value was $: strconv.ParseInt: parsing \"$\": invalid syntax"),
		errors.New("failed to parse int64 for SapnetweaverSessionsHTTPCount, value was $: strconv.ParseInt: parsing \"$\": invalid syntax"),
		errors.New("failed to parse int64 for SapnetweaverSessionsSecurityCount, value was $: strconv.ParseInt: parsing \"$\": invalid syntax"),
		errors.New("failed to parse int64 for SapnetweaverWorkProcessesActiveCount, value was $: strconv.ParseInt: parsing \"$\": invalid syntax"),
		errors.New("failed to parse int64 for SapnetweaverSessionsWebCount, value was $: strconv.ParseInt: parsing \"$\": invalid syntax"),
		errors.New("failed to parse int64 for SapnetweaverSessionsBrowserCount, value was $: strconv.ParseInt: parsing \"$\": invalid syntax"),
		errors.New("failed to parse int64 for SapnetweaverSessionsEjbCount, value was $: strconv.ParseInt: parsing \"$\": invalid syntax"),
		errors.New("failed to collect metric ICM: invalid control state color value"),
		errors.New("failed to parse int64 for SapnetweaverHostSpoolListUsed, value was $: strconv.ParseInt: parsing \"$\": invalid syntax"),
		errors.New("failed to parse int64 for SapnetweaverShortDumpsRate, value was $: strconv.ParseInt: parsing \"$\": invalid syntax"),
	), err.Error())

	require.Error(t, err)
	require.Equal(t, 1, actualMetrics.ResourceMetrics().Len())
	require.Equal(t, 1, actualMetrics.DataPointCount())
	require.Equal(t, 1, actualMetrics.MetricCount())

	require.EqualValues(t, "", scraper.hostname)
	require.EqualValues(t, "", scraper.instance)
}

func TestScraperScrapeAPIError(t *testing.T) {
	mockService := mocks.MockWebService{}
	mockService.On("GetAlertTree").Return(nil, errors.New("unexpected error"))
	mockService.On("EnqGetLockTable").Return(nil, errors.New("unexpected error"))
	mockService.On("GetInstanceProperties").Return(nil, errors.New("unexpected error"))

	cfg := createDefaultConfig().(*Config)
	cfg.Endpoint = defaultEndpoint
	cfg.Username = "root"
	cfg.Password = "password"

	testClient, err := newSoapClient(cfg, componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	scraper := newSapNetweaverScraper(receivertest.NewNopCreateSettings(), createDefaultConfig().(*Config))
	scraper.service = &mockService
	scraper.client = testClient

	actualMetrics, err := scraper.scrape(context.Background())
	require.NotNil(t, err)

	require.Equal(t, 0, actualMetrics.ResourceMetrics().Len())
	require.Equal(t, 0, actualMetrics.DataPointCount())
	require.Equal(t, 0, actualMetrics.MetricCount())

	require.EqualError(t, multierr.Combine(
		errors.New("failed to get current instance details: unexpected error"),
		errors.New("failed to collect Alert Tree metrics: unexpected error"),
		errors.New("failed to collect Enq Lock Table metrics: unexpected error"),
	), err.Error())
}

func TestScraperScrapeEmptyXML(t *testing.T) {
	mockService := mocks.MockWebService{}
	mockService.On("EnqGetStatistic").Return(nil, nil)
	mockService.On("GetAlertTree").Return(&models.GetAlertTreeResponse{}, nil)
	mockService.On("EnqGetLockTable").Return(&models.EnqGetLockTableResponse{}, nil)
	mockService.On("GetInstanceProperties").Return(&models.GetInstancePropertiesResponse{}, nil)

	cfg := createDefaultConfig().(*Config)
	cfg.Endpoint = defaultEndpoint
	cfg.Username = "root"
	cfg.Password = "password"

	testClient, err := newSoapClient(cfg, componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	scraper := newSapNetweaverScraper(receivertest.NewNopCreateSettings(), createDefaultConfig().(*Config))
	scraper.service = &mockService
	scraper.client = testClient

	actualMetrics, err := scraper.scrape(context.Background())
	require.NotNil(t, err)

	require.Equal(t, 1, actualMetrics.ResourceMetrics().Len())
	require.Equal(t, 1, actualMetrics.DataPointCount())
	require.Equal(t, 1, actualMetrics.MetricCount())

	require.EqualError(t, multierr.Combine(
		errors.New("failed to collect metric CPU_Utilization: value not found"),
		errors.New("failed to collect metric Memory Overhead: value not found"),
		errors.New("failed to collect metric Memory Swapped Out: value not found"),
		errors.New("failed to collect metric CurrentHttpSessions: value not found"),
		errors.New("failed to collect metric CurrentSecuritySessions: value not found"),
		errors.New("failed to collect metric Total Number of Work Processes: value not found"),
		errors.New("failed to collect metric Web Sessions: value not found"),
		errors.New("failed to collect metric Browser Sessions: value not found"),
		errors.New("failed to collect metric EJB Sessions: value not found"),
		errors.New("failed to collect metric ICM: value not found"),
		errors.New("failed to collect metric HostspoolListUsed: value not found"),
		errors.New("failed to collect metric Shortdumps Frequency: value not found"),
	), err.Error())
}

func TestParseResponseTypes(t *testing.T) {
	testCases := []struct {
		desc          string
		rawValue      string
		expectedValue string
	}{
		{
			desc:          "rate case",
			rawValue:      "40 /min",
			expectedValue: "40",
		},
		{
			desc:          "byte case",
			rawValue:      "40 MB",
			expectedValue: "40",
		},
		{
			desc:          "percentage case",
			rawValue:      "40 %",
			expectedValue: "40",
		},
		{
			desc:          "hypen case",
			rawValue:      "- %",
			expectedValue: "-",
		},
		{
			desc:          "empty case",
			rawValue:      "",
			expectedValue: "",
		},
		{
			desc:          "only value case",
			rawValue:      "40",
			expectedValue: "40",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			value := strings.Split(tc.rawValue, " ")
			require.EqualValues(t, tc.expectedValue, value[0])
		})
	}
}

func loadAPIResponseData(t *testing.T, folder, fileName string) []byte {
	t.Helper()
	fullPath := filepath.Join("testdata", folder, fileName)

	data, err := os.ReadFile(fullPath)
	require.NoError(t, err)

	return data
}
