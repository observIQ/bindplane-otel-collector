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

package m365receiver // import "github.com/observiq/observiq-otel-collector/receiver/m365receiver"

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/observiq/observiq-otel-collector/receiver/m365receiver/internal/metadata"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestBadToken(t *testing.T) {
	//testing error handling at start of scrape
	//mocks
	mc := &mockClient{}
	root := "https://graph.microsoft.com/v1.0/reports/"
	scraper := newM365Scraper(
		receivertest.NewNopCreateSettings(),
		&Config{MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig()},
	)
	scraper.client = mc

	//test 1: incorrect token requirements, scraper fails to gen a new, correct token
	mc.On("GetCSV", mock.Anything, root+"getSharePointSiteUsageFileCounts(period='D7')").Return([]string{}, fmt.Errorf("access token invalid")).Once()
	mc.On("GetToken", mock.Anything).Return(fmt.Errorf("the provided client_id is incorrect or does not exist within the given tenant directory")).Once()

	_, err := scraper.scrape(context.Background())
	require.EqualError(t, err, "the provided client_id is incorrect or does not exist within the given tenant directory")

	//test 2: stale token, getCSV will return empty data just for simplicity in this test
	mc.On("GetCSV", mock.Anything, root+"getSharePointSiteUsageFileCounts(period='D7')").Return([]string{}, fmt.Errorf("access token invalid")).Once()
	mc.On("GetToken", mock.Anything).Return(nil).Once()
	mc.On("GetCSV", mock.Anything, root+"getSharePointSiteUsageFileCounts(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getSharePointSiteUsageSiteCounts(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getSharePointSiteUsagePages(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getSharePointActivityPages(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getSharePointSiteUsageStorage(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getTeamsDeviceUsageDistributionUserCounts(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getTeamsUserActivityCounts(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getOneDriveUsageFileCounts(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getOneDriveActivityUserCounts(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getMailboxUsageMailboxCounts(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getEmailActivityCounts(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getMailboxUsageStorage(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getEmailAppUsageAppsUserCounts(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getMailboxUsageQuotaStatusMailboxCounts(period='D7')").Return([]string{}, nil)

	_, err = scraper.scrape(context.Background())
	require.NoError(t, err)
}

func TestPartialMetrics(t *testing.T) {
	//mocks, only do the first endpoint, leave out all other metrics
	root := "https://graph.microsoft.com/v1.0/reports/"
	mc := &mockClient{}
	mc.On("GetCSV", mock.Anything, root+"getSharePointSiteUsageFileCounts(period='D7')").Return([]string{
		"2023-04-23", "All", "2", "0", "2023-04-23", "7",
	}, nil)
	mc.On("GetCSV", mock.Anything, root+"getSharePointSiteUsageSiteCounts(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getSharePointSiteUsagePages(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getSharePointActivityPages(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getSharePointSiteUsageStorage(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getTeamsDeviceUsageDistributionUserCounts(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getTeamsUserActivityCounts(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getOneDriveUsageFileCounts(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getOneDriveActivityUserCounts(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getMailboxUsageMailboxCounts(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getEmailActivityCounts(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getMailboxUsageStorage(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getEmailAppUsageAppsUserCounts(period='D7')").Return([]string{}, nil)
	mc.On("GetCSV", mock.Anything, root+"getMailboxUsageQuotaStatusMailboxCounts(period='D7')").Return([]string{}, nil)

	scraper := newM365Scraper(
		receivertest.NewNopCreateSettings(),
		&Config{MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig()},
	)

	scraper.client = mc

	actualMetrics, err := scraper.scrape(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, actualMetrics)

	//generate testdata file
	// m := pmetric.JSONMarshaler{}
	// mBytes, err := m.MarshalMetrics(actualMetrics)
	// require.NoError(t, err)
	// goldenPath := filepath.Join("testdata", "metrics", "unit-test-partialMetrics.json")
	// err = os.WriteFile(goldenPath, mBytes, 0666)
	// require.NoError(t, err)

	//validate output of scrape
	expectedFile := filepath.Join("testdata", "metrics", "unit-test-partialMetrics.json")
	expectedMetrics, err := ReadMetrics(expectedFile)
	require.NoError(t, err)
	require.NoError(t, pmetrictest.CompareMetrics(expectedMetrics, actualMetrics,
		pmetrictest.IgnoreMetricDataPointsOrder(), pmetrictest.IgnoreStartTimestamp(), pmetrictest.IgnoreTimestamp()),
	)
}

func TestScraper(t *testing.T) {
	//mocks
	root := "https://graph.microsoft.com/v1.0/reports/"
	mc := &mockClient{}
	mc.On("GetCSV", mock.Anything, root+"getSharePointSiteUsageFileCounts(period='D7')").Return([]string{
		"2023-04-23", "All", "2", "0", "2023-04-23", "7",
	}, nil)
	mc.On("GetCSV", mock.Anything, root+"getSharePointSiteUsageSiteCounts(period='D7')").Return([]string{
		"2023-04-23", "All", "8", "0", "2023-04-23", "7",
	}, nil)
	mc.On("GetCSV", mock.Anything, root+"getSharePointSiteUsagePages(period='D7')").Return([]string{
		"2023-04-23", "All", "3", "2023-04-23", "7",
	}, nil)
	mc.On("GetCSV", mock.Anything, root+"getSharePointActivityPages(period='D7')").Return([]string{
		"2023-04-23", "10", "2023-04-23", "7",
	}, nil)
	mc.On("GetCSV", mock.Anything, root+"getSharePointSiteUsageStorage(period='D7')").Return([]string{
		"2023-04-23", "All", "1111", "2023-04-23", "7",
	}, nil)
	mc.On("GetCSV", mock.Anything, root+"getTeamsDeviceUsageDistributionUserCounts(period='D7')").Return([]string{
		"2023-04-23", "2", "0", "4", "6", "8", "10", "12", "14", "7",
	}, nil)
	mc.On("GetCSV", mock.Anything, root+"getTeamsUserActivityCounts(period='D7')").Return([]string{
		"2023-04-23", "2023-04-23", "2", "1", "1", "4", "6", "8", "1", "1", "1", "1", "1", "7",
	}, nil)
	mc.On("GetCSV", mock.Anything, root+"getOneDriveUsageFileCounts(period='D7')").Return([]string{
		"2023-04-23", "All", "6", "3", "2024-04-23", "7",
	}, nil)
	mc.On("GetCSV", mock.Anything, root+"getOneDriveActivityUserCounts(period='D7')").Return([]string{
		"2023-04-23", "2", "4", "6", "8", "2023-04-23", "7",
	}, nil)
	mc.On("GetCSV", mock.Anything, root+"getMailboxUsageMailboxCounts(period='D7')").Return([]string{
		"2023-04-23", "5", "3", "2023-04-23", "7",
	}, nil)
	mc.On("GetCSV", mock.Anything, root+"getEmailActivityCounts(period='D7')").Return([]string{
		"2023-04-23", "2", "4", "6", "1", "1", "2023-04-23", "7",
	}, nil)
	mc.On("GetCSV", mock.Anything, root+"getMailboxUsageStorage(period='D7')").Return([]string{
		"2023-04-23", "50", "2023-04-23", "7",
	}, nil)
	mc.On("GetCSV", mock.Anything, root+"getEmailAppUsageAppsUserCounts(period='D7')").Return([]string{
		"2023-04-23", "1", "2", "4", "6", "8", "10", "12", "14", "16", "2023-04-23", "7",
	}, nil)
	mc.On("GetCSV", mock.Anything, root+"getMailboxUsageQuotaStatusMailboxCounts(period='D7')").Return([]string{
		"2023-04-23", "2", "4", "6", "8", "10", "2023-04-23", "7",
	}, nil)

	scraper := newM365Scraper(
		receivertest.NewNopCreateSettings(),
		&Config{MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig()},
	)

	scraper.client = mc

	actualMetrics, err := scraper.scrape(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, actualMetrics)

	//generate testdata file
	// m := pmetric.JSONMarshaler{}
	// mBytes, err := m.MarshalMetrics(actualMetrics)
	// require.NoError(t, err)
	// goldenPath := filepath.Join("testdata", "metrics", "unit-test-metrics.json")
	// err = os.WriteFile(goldenPath, mBytes, 0666)
	// require.NoError(t, err)

	//validate output of scrape
	expectedFile := filepath.Join("testdata", "metrics", "unit-test-metrics.json")
	expectedMetrics, err := ReadMetrics(expectedFile)
	require.NoError(t, err)
	require.NoError(t, pmetrictest.CompareMetrics(expectedMetrics, actualMetrics,
		pmetrictest.IgnoreMetricDataPointsOrder(), pmetrictest.IgnoreStartTimestamp(), pmetrictest.IgnoreTimestamp()),
	)
}

type mockClient struct {
	mock.Mock
}

func (mw *mockClient) GetCSV(ctx context.Context, endpoint string) ([]string, error) {
	args := mw.Called(ctx, endpoint)
	return args.Get(0).([]string), args.Error(1)
}

func (mw *mockClient) GetToken(ctx context.Context) error {
	args := mw.Called(ctx)
	return args.Error(0)
}

func (mw *mockClient) shutdown() error {
	return nil
}

func ReadMetrics(filePath string) (pmetric.Metrics, error) {
	expectedFileBytes, err := os.ReadFile(filePath)
	if err != nil {
		return pmetric.Metrics{}, err
	}
	unmarshaller := &pmetric.JSONUnmarshaler{}
	return unmarshaller.UnmarshalMetrics(expectedFileBytes)
}
