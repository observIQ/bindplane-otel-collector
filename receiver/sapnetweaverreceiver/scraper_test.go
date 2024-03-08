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

package sapnetweaverreceiver // import "github.com/observiq/bindplane-agent/receiver/sapnetweaverreceiver"

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/config/configtls"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/multierr"

	"github.com/observiq/bindplane-agent/receiver/sapnetweaverreceiver/internal/mocks"
	"github.com/observiq/bindplane-agent/receiver/sapnetweaverreceiver/internal/models"
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
					ClientConfig: confighttp.ClientConfig{
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
					ClientConfig: confighttp.ClientConfig{
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
	alertTreeResponseData := loadAPIResponseData(t, "api-responses", "AlertTreeResponse.xml")
	var alertTreeResponse *models.GetAlertTreeResponse
	err := xml.Unmarshal(alertTreeResponseData, &alertTreeResponse)
	require.NoError(t, err)

	abapSystemWpTabledata := loadAPIResponseData(t, "api-responses", "ABAPSystemWPTableResponse.xml")
	var abapSystemWpTableResponse *models.ABAPGetSystemWPTableResponse
	err = xml.Unmarshal(abapSystemWpTabledata, &abapSystemWpTableResponse)
	require.NoError(t, err)

	enqStatisticData := loadAPIResponseData(t, "api-responses", "EnqStatisticResponse.xml")
	var enqStatisticResponse *models.EnqGetStatisticResponse
	err = xml.Unmarshal(enqStatisticData, &enqStatisticResponse)
	require.NoError(t, err)

	processListData := loadAPIResponseData(t, "api-responses", "ProcessListResponse.xml")
	var processListResponse *models.GetProcessListResponse
	err = xml.Unmarshal(processListData, &processListResponse)
	require.NoError(t, err)

	queueStatisticData := loadAPIResponseData(t, "api-responses", "QueueStatisticResponse.xml")
	var queueStatisticResponse *models.GetQueueStatisticResponse
	err = xml.Unmarshal(queueStatisticData, &queueStatisticResponse)
	require.NoError(t, err)

	systemInstanceListData := loadAPIResponseData(t, "api-responses", "SystemInstanceListResponse.xml")
	var systemInstanceListResponse *models.GetSystemInstanceListResponse
	err = xml.Unmarshal(systemInstanceListData, &systemInstanceListResponse)
	require.NoError(t, err)

	InstancePropertiesData := loadAPIResponseData(t, "api-responses", "InstancePropertiesResponse.xml")
	var InstancePropertiesResponse *models.GetInstancePropertiesResponse
	err = xml.Unmarshal(InstancePropertiesData, &InstancePropertiesResponse)
	require.NoError(t, err)

	certificate1 := processFile(string(loadAPIResponseData(t, "api-responses", "certificate1.txt")))
	certificate2 := processFile(string(loadAPIResponseData(t, "api-responses", "certificate2.txt")))
	rfcConnections := string(loadAPIResponseData(t, "api-responses", "dpmon-c-rfc-connections.txt"))
	sessionsTable := string(loadAPIResponseData(t, "api-responses", "dpmon-v-sessions-table.txt"))

	mockService := mocks.MockWebService{}
	mockService.On("GetAlertTree").Return(alertTreeResponse, nil)
	mockService.On("ABAPGetSystemWPTable").Return(abapSystemWpTableResponse, nil)
	mockService.On("EnqGetStatistic").Return(enqStatisticResponse, nil)
	mockService.On("GetProcessList").Return(processListResponse, nil)
	mockService.On("GetQueueStatistic").Return(queueStatisticResponse, nil)
	mockService.On("GetSystemInstanceList").Return(systemInstanceListResponse, nil)
	mockService.On("GetInstanceProperties").Return(InstancePropertiesResponse, nil)
	mockService.On("FindFile", "-L", "/usr/sap", "-name", "*.pse").Return([]string{"/usr/sap/EPP/D00/sec/SAPSSLA.pse", "/usr/sap/EPP/D00/sec/SAPSSLC.pse"}, nil)
	mockService.On("CertExecute", "/usr/sap/hostctrl/exe/sapgenpse get_my_name -p /usr/sap/EPP/D00/sec/SAPSSLA.pse -n validity").Return(certificate1, nil)
	mockService.On("CertExecute", "/usr/sap/hostctrl/exe/sapgenpse get_my_name -p /usr/sap/EPP/D00/sec/SAPSSLC.pse -n validity").Return(certificate2, nil)
	mockService.On("FindFile", "-L", "/usr/sap", "-name", "*.pse").Return([]string{""}, nil)
	mockService.On("FindFile", "-L", "/usr/sap", "-name", "dpmon", "-path", "*/exe/dpmon").Return([]string{"/usr/sap/EPP/D00/exe/dpmon"}, nil)
	mockService.On("DpmonExecute", "echo q | /usr/sap/EPP/D00/exe/dpmon pf=/sapmnt/EPP/profile/EPP_D00_sap-app-1 c").Return(rfcConnections, nil)
	mockService.On("DpmonExecute", "echo q | /usr/sap/EPP/D00/exe/dpmon pf=/sapmnt/EPP/profile/EPP_D00_sap-app-1 v").Return(sessionsTable, nil)

	cfg := createDefaultConfig().(*Config)
	cfg.Endpoint = defaultEndpoint
	cfg.Username = "root"
	cfg.Password = "password"
	cfg.Profile = "/sapmnt/EPP/profile/EPP_D00_sap-app-1"

	testClient, err := newSoapClient(cfg, componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	scraper := newSapNetweaverScraper(receivertest.NewNopCreateSettings(), cfg)
	scraper.service = &mockService
	scraper.client = testClient

	actualMetrics, err := scraper.scrape(context.Background())
	require.NoError(t, err)

	// Uncomment to generate golden file.
	// WriteMetrics(t, filepath.Join("testdata", "golden-response", "expected.json"), actualMetrics)

	expected, err := ReadMetrics(filepath.Join("testdata", "golden-response", "expected.json"))
	require.NoError(t, err)

	require.NoError(t, pmetrictest.CompareMetrics(expected, actualMetrics, pmetrictest.IgnoreMetricDataPointsOrder(),
		pmetrictest.IgnoreStartTimestamp(), pmetrictest.IgnoreTimestamp(), pmetrictest.IgnoreMetricValues()))

}

func TestScraperScrapeEmpty(t *testing.T) {

	alertTreeResponseDataEmpty := loadAPIResponseData(t, "api-responses", "AlertTreeEmptyResponse.xml")
	var alertTreeResponse *models.GetAlertTreeResponse
	err := xml.Unmarshal(alertTreeResponseDataEmpty, &alertTreeResponse)
	require.NoError(t, err)

	abapSystemWpTabledataEmpty := loadAPIResponseData(t, "api-responses", "ABAPSystemWPTableEmptyResponse.xml")
	var abapSystemWpTableResponse *models.ABAPGetSystemWPTableResponse
	err = xml.Unmarshal(abapSystemWpTabledataEmpty, &abapSystemWpTableResponse)
	require.NoError(t, err)

	enqStatisticDataEmpty := loadAPIResponseData(t, "api-responses", "EnqStatisticEmptyResponse.xml")
	var enqStatisticResponse *models.EnqGetStatisticResponse
	err = xml.Unmarshal(enqStatisticDataEmpty, &enqStatisticResponse)
	require.NoError(t, err)

	processListDataEmpty := loadAPIResponseData(t, "api-responses", "ProcessListEmptyResponse.xml")
	var processListResponse *models.GetProcessListResponse
	err = xml.Unmarshal(processListDataEmpty, &processListResponse)
	require.NoError(t, err)

	queueStatisticDataEmpty := loadAPIResponseData(t, "api-responses", "QueueStatisticEmptyResponse.xml")
	var queueStatisticResponse *models.GetQueueStatisticResponse
	err = xml.Unmarshal(queueStatisticDataEmpty, &queueStatisticResponse)
	require.NoError(t, err)

	systemInstanceListDataEmpty := loadAPIResponseData(t, "api-responses", "SystemInstanceListEmptyResponse.xml")
	var systemInstanceListResponse *models.GetSystemInstanceListResponse
	err = xml.Unmarshal(systemInstanceListDataEmpty, &systemInstanceListResponse)
	require.NoError(t, err)

	InstancePropertiesDataEmpty := loadAPIResponseData(t, "api-responses", "InstancePropertiesEmptyResponse.xml")
	var InstancePropertiesResponse *models.GetInstancePropertiesResponse
	err = xml.Unmarshal(InstancePropertiesDataEmpty, &InstancePropertiesResponse)
	require.NoError(t, err)

	mockService := mocks.MockWebService{}
	mockService.On("GetAlertTree").Return(alertTreeResponse, nil)
	mockService.On("ABAPGetSystemWPTable").Return(abapSystemWpTableResponse, nil)
	mockService.On("EnqGetStatistic").Return(enqStatisticResponse, nil)
	mockService.On("GetProcessList").Return(processListResponse, nil)
	mockService.On("GetQueueStatistic").Return(queueStatisticResponse, nil)
	mockService.On("GetSystemInstanceList").Return(systemInstanceListResponse, nil)
	mockService.On("GetInstanceProperties").Return(InstancePropertiesResponse, nil)
	mockService.On("FindFile", "-L", "/usr/sap", "-name", "*.pse").Return([]string{""}, nil)
	mockService.On("FindFile", "-L", "/usr/sap", "-name", "dpmon", "-path", "*/exe/dpmon").Return([]string{"/usr/sap/EPP/D00/exe/dpmon"}, nil)
	mockService.On("DpmonExecute", "echo q | /usr/sap/EPP/D00/exe/dpmon pf=/sapmnt/EPP/profile/EPP_D00_sap-app-1 c").Return("", nil)
	mockService.On("DpmonExecute", "echo q | /usr/sap/EPP/D00/exe/dpmon pf=/sapmnt/EPP/profile/EPP_D00_sap-app-1 v").Return("", nil)

	cfg := createDefaultConfig().(*Config)
	cfg.Endpoint = defaultEndpoint
	cfg.Username = "root"
	cfg.Password = "password"
	cfg.Profile = "/sapmnt/EPP/profile/EPP_D00_sap-app-1"

	testClient, err := newSoapClient(cfg, componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	scraper := newSapNetweaverScraper(receivertest.NewNopCreateSettings(), cfg)
	scraper.service = &mockService
	scraper.client = testClient

	actualMetrics, err := scraper.scrape(context.Background())
	require.Error(t, err)

	require.EqualError(t, multierr.Combine(
		errors.New("failed to collect metric DBRequestTime: value not found"),
		errors.New("failed to collect metric CPU_Utilization: value not found"),
		errors.New("failed to collect metric System Utilization: value not found"),
		errors.New("failed to collect metric ErrorsInWpSPO: value not found"),
		errors.New("failed to collect metric AbortedJobs: value not found"),
		errors.New("failed to collect metric Swap_Space_Percentage_Used: value not found"),
		errors.New("failed to collect metric Configured Memory: value not found"),
		errors.New("failed to collect metric Free Memory: value not found"),
		errors.New("failed to collect metric Number of Sessions: value not found"),
		errors.New("failed to collect metric AbapErrorInUpdate: value not found"),
		errors.New("failed to collect metric ResponseTimeDialog with attribute dialog: value not found"),
		errors.New("failed to collect metric ResponseTimeDialogRFC with attribute dialogRFC: value not found"),
		errors.New("failed to collect metric ResponseTime(StandardTran.) with attribute transaction: value not found"),
		errors.New("failed to collect metric ResponseTimeHTTP with attribute http: value not found"),
		errors.New("failed to collect metric StatNoOfRequests: value not found"),
		errors.New("failed to collect metric StatNoOfTimeouts: value not found"),
		errors.New("failed to collect metric StatNoOfConnectErrors: value not found"),
		errors.New("failed to collect metric EvictedEntries: value not found"),
		errors.New("failed to collect metric CacheHits: value not found"),
		errors.New("failed to collect metric HostspoolListUsed: value not found"),
		errors.New("failed to collect metric Shortdumps Frequency: value not found"),
		errors.New("failed to collect metric Memory Overhead: value not found"),
		errors.New("failed to collect metric Memory Swapped Out: value not found"),
		errors.New("failed to collect metric CurrentHttpSessions: value not found"),
		errors.New("failed to collect metric CurrentSecuritySessions: value not found"),
		errors.New("failed to collect metric Web Sessions: value not found"),
		errors.New("failed to collect metric Browser Sessions: value not found"),
		errors.New("failed to collect metric EJB Sessions: value not found"),
		errors.New("failed to collect metric Active Work Processes: value not found"),
		errors.New("failed to collect metric LocksNow: value not found"),
		errors.New("failed to collect metric LocksHigh: value not found"),
		errors.New("failed to collect metric LocksMax: value not found"),
		errors.New("failed to collect metric DequeueErrors: value not found"),
		errors.New("failed to collect metric EnqueueErrors: value not found"),
		errors.New("failed to collect metric LockTime: value not found"),
		errors.New("failed to collect metric LockWaitTime: value not found"),
		errors.New("failed to collect metric Queue count, peak and max: value not found"),
		errors.New("failed to collect metric Process Availability: value not found"),
		errors.New("failed to collect metric Service Availability: value not found"),
	), err.Error())

	expected, err := ReadMetrics(filepath.Join("testdata", "golden-response", "empty-expected.json"))
	require.NoError(t, err)

	require.NoError(t, pmetrictest.CompareMetrics(expected, actualMetrics, pmetrictest.IgnoreMetricDataPointsOrder(),
		pmetrictest.IgnoreStartTimestamp(), pmetrictest.IgnoreTimestamp()))
}

func TestScraperScrapeAPIError(t *testing.T) {
	mockService := mocks.MockWebService{}
	mockService.On("GetAlertTree").Return(nil, errors.New("unexpected error"))
	mockService.On("ABAPGetSystemWPTable").Return(nil, errors.New("unexpected error"))
	mockService.On("EnqGetStatistic").Return(nil, errors.New("unexpected error"))
	mockService.On("GetProcessList").Return(nil, errors.New("unexpected error"))
	mockService.On("GetQueueStatistic").Return(nil, errors.New("unexpected error"))
	mockService.On("GetSystemInstanceList").Return(nil, errors.New("unexpected error"))
	mockService.On("GetInstanceProperties").Return(nil, errors.New("unexpected error"))
	mockService.On("FindFile", "-L", "/usr/sap", "-name", "*.pse").Return([]string{}, errors.New("unexpected error"))
	mockService.On("FindFile", "-L", "/usr/sap", "-name", "dpmon", "-path", "*/exe/dpmon").Return([]string{}, errors.New("unexpected error"))

	cfg := createDefaultConfig().(*Config)
	cfg.Endpoint = defaultEndpoint
	cfg.Username = "root"
	cfg.Password = "password"
	cfg.Profile = "/sapmnt/EPP/profile/EPP_D00_sap-app-1"

	testClient, err := newSoapClient(cfg, componenttest.NewNopHost(), componenttest.NewNopTelemetrySettings())
	require.NoError(t, err)

	scraper := newSapNetweaverScraper(receivertest.NewNopCreateSettings(), cfg)
	scraper.service = &mockService
	scraper.client = testClient

	actualMetrics, err := scraper.scrape(context.Background())
	require.NotNil(t, err)

	require.Equal(t, 0, actualMetrics.ResourceMetrics().Len())
	require.Equal(t, 0, actualMetrics.DataPointCount())
	require.Equal(t, 0, actualMetrics.MetricCount())

	require.EqualError(t, multierr.Combine(
		errors.New("failed to collect GetInstanceProperties metrics: unexpected error"),
		errors.New("failed to collect Alert Tree metrics: unexpected error"),
		errors.New("failed to collect ABAPGetSystemWPTable metrics: unexpected error"),
		errors.New("failed to collect EnqGetStatistic metrics: unexpected error"),
		errors.New("failed to collect GetQueueStatistic metrics: unexpected error"),
		errors.New("failed to collect GetProcessList metrics: unexpected error"),
		errors.New("failed to collect GetSystemInstanceList metrics: unexpected error"),
		errors.New("failed to find certificate files: unexpected error"),
		errors.New("failed find dpmon executable: unexpected error"),
	), err.Error())
}

func TestParseDpmonRFCConnections(t *testing.T) {
	t.Run("empty rfc table that contains Communication Table is empty", func(t *testing.T) {
		rfcConnectionsEmpty := string(loadAPIResponseData(t, "api-responses", "dpmon-c-rfc-connections-empty.txt"))
		rfcConnectionMap := parseRfcConnectionsTable(rfcConnectionsEmpty)
		require.Empty(t, rfcConnectionMap)
	})

	t.Run("rfc table with values", func(t *testing.T) {
		rfcConnections := string(loadAPIResponseData(t, "api-responses", "dpmon-c-rfc-connections.txt"))
		rfcConnectionMap := parseRfcConnectionsTable(rfcConnections)

		require.EqualValues(t, int64(2), rfcConnectionMap["HTTP"])
		require.EqualValues(t, int64(2), rfcConnectionMap["CPIC"])
	})
}

func TestParseSessions(t *testing.T) {
	t.Run("empty table", func(t *testing.T) {
		sessionsTableEmpty := string(loadAPIResponseData(t, "api-responses", "dpmon-v-sessions-table-empty.txt"))
		sessionsTableMap := parseSessionTable(sessionsTableEmpty)
		require.Empty(t, sessionsTableMap)
	})

	t.Run("session table with values", func(t *testing.T) {
		sessionsTable := string(loadAPIResponseData(t, "api-responses", "dpmon-v-sessions-table.txt"))
		sessionsTableMap := parseSessionTable(sessionsTable)
		require.EqualValues(t, int64(2), sessionsTableMap["RFC_UI"])
		require.EqualValues(t, int64(1), sessionsTableMap["HTTP"])
		require.EqualValues(t, int64(1), sessionsTableMap["BATCH"])
	})
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

// ReadMetrics reads a pmetric.Metrics from the specified file
func ReadMetrics(filePath string) (pmetric.Metrics, error) {
	expectedFileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return pmetric.Metrics{}, err
	}
	unmarshaller := &pmetric.JSONUnmarshaler{}
	return unmarshaller.UnmarshalMetrics(expectedFileBytes)
}

// WriteMetrics writes a pmetric.Metrics to the specified file
func WriteMetrics(t *testing.T, filePath string, metrics pmetric.Metrics) error {
	if err := writeMetrics(filePath, metrics); err != nil {
		return err
	}
	t.Logf("Golden file successfully written to %s.", filePath)
	t.Log("NOTE: The WriteMetrics call must be removed in order to pass the test.")
	t.Fail()
	return nil
}

func writeMetrics(filePath string, metrics pmetric.Metrics) error {
	unmarshaler := &pmetric.JSONMarshaler{}
	fileBytes, err := unmarshaler.MarshalMetrics(metrics)
	if err != nil {
		return err
	}
	var jsonVal map[string]interface{}
	if err = json.Unmarshal(fileBytes, &jsonVal); err != nil {
		return err
	}
	b, err := json.MarshalIndent(jsonVal, "", "   ")
	if err != nil {
		return err
	}
	b = append(b, []byte("\n")...)
	return os.WriteFile(filePath, b, 0600)
}
