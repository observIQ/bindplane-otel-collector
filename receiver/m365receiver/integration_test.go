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

package m365receiver // import "github.com/observiq/bindplane-agent/receiver/m365receiver"

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/pdatatest/pmetrictest"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestM365Integration(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)

	//set required config values
	cfg.TenantID = "testTenantID"
	cfg.ClientID = "testClientID"
	cfg.ClientSecret = "testClientSecret"

	//create receiver
	settings := receivertest.NewNopCreateSettings()
	rcvr := newM365Scraper(settings, cfg)

	//create m365Client object with the http.Client = to the mock server for the integration tests
	mockServer := newIntMockServer()
	client := newM365Client(mockServer.Client(), cfg, "https://graph.microsoft.com/.default")
	client.authEndpoint = mockServer.URL + "/" + cfg.TenantID
	err := client.GetToken(context.Background())
	require.NoError(t, err)
	rcvr.client = client
	rcvr.root = mockServer.URL + "/"

	actualMetrics, err := rcvr.scrape(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, actualMetrics)

	//generate testdata file
	// m := pmetric.JSONMarshaler{}
	// mBytes, err := m.MarshalMetrics(actualMetrics)
	// require.NoError(t, err)
	// path := filepath.Join("testdata", "metrics", "integration-test-metrics.json")
	// err = os.WriteFile(path, mBytes, 0666)
	// require.NoError(t, err)

	//check output
	expectedFile := filepath.Join("testdata", "metrics", "integration-test-metrics.json")
	expectedMetrics, err := ReadMetrics(expectedFile)
	require.NoError(t, err)

	require.NoError(t, pmetrictest.CompareMetrics(expectedMetrics, actualMetrics,
		pmetrictest.IgnoreMetricValues(), pmetrictest.IgnoreMetricDataPointsOrder(),
		pmetrictest.IgnoreStartTimestamp(), pmetrictest.IgnoreTimestamp()))
}

func newIntMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(httpTestHandler))
}

func checkMethodAndAuth(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != "GET" {
		rw.WriteHeader(400)
		return fmt.Errorf("err")
	}
	if a := req.Header.Get("Authorization"); a != "Bearer testAccessToken" {
		rw.WriteHeader(400)
		rw.Write([]byte(`{"error": {"code": "InvalidAuthenticationToken"}}`))
		return fmt.Errorf("err")
	}
	return nil
}

func httpTestHandler(rw http.ResponseWriter, req *http.Request) {
	//token authorization
	if req.URL.String() == "/testTenantID" {
		if req.Method != "POST" {
			rw.WriteHeader(400)
			return
		}

		req.ParseForm()
		if gType := req.Form.Get("grant_type"); gType != "client_credentials" {
			rw.WriteHeader(400)
			rw.Write([]byte("Error, incorrect grant_type"))
			return
		}
		if scope := req.Form.Get("scope"); scope != "https://graph.microsoft.com/.default" {
			rw.WriteHeader(400)
			rw.Write([]byte("Error, incorrect scope"))
			return
		}
		if cID := req.Form.Get("client_id"); cID != "testClientID" {
			rw.WriteHeader(400)
			rw.Write([]byte(`{"error":"unauthorized_client"}`))
			return
		}
		if cSec := req.Form.Get("client_secret"); cSec != "testClientSecret" {
			rw.WriteHeader(401)
			rw.Write([]byte(`{"error": "invalid_client"}`))
			return
		}

		rw.WriteHeader(200)
		rw.Write([]byte(
			`{
				"token_type": "Bearer",
				"expires_in": 3599,
				"ext_expires_in": 3599,
				"access_token": "testAccessToken"
			}`,
		))
		return
	}

	/////////////		Endpoints
	//sharepoint
	if req.URL.String() == "/getSharePointSiteUsageFileCounts(period='D7')" {
		if err := checkMethodAndAuth(rw, req); err != nil {
			return
		}
		rw.Header().Add("Content-Type", "text/plain")
		rw.Header().Add("Location", "/SharePointSiteUsageFileCountsCSV")
		rw.WriteHeader(302)
		return
	}
	if req.URL.String() == "/getSharePointSiteUsageSiteCounts(period='D7')" {
		if err := checkMethodAndAuth(rw, req); err != nil {
			return
		}
		rw.Header().Add("Content-Type", "text/plain")
		rw.Header().Add("Location", "/SharePointSiteUsageSiteCountsCSV")
		rw.WriteHeader(302)
		return
	}
	if req.URL.String() == "/getSharePointSiteUsagePages(period='D7')" {
		if err := checkMethodAndAuth(rw, req); err != nil {
			return
		}
		rw.Header().Add("Content-Type", "text/plain")
		rw.Header().Add("Location", "/SharePointSiteUsagePagesCSV")
		rw.WriteHeader(302)
		return
	}
	if req.URL.String() == "/getSharePointActivityPages(period='D7')" {
		if err := checkMethodAndAuth(rw, req); err != nil {
			return
		}
		rw.Header().Add("Content-Type", "text/plain")
		rw.Header().Add("Location", "/SharePointActivityPagesCSV")
		rw.WriteHeader(302)
		return
	}
	if req.URL.String() == "/getSharePointSiteUsageStorage(period='D7')" {
		if err := checkMethodAndAuth(rw, req); err != nil {
			return
		}
		rw.Header().Add("Content-Type", "text/plain")
		rw.Header().Add("Location", "/SharePointSiteUsageStorageCSV")
		rw.WriteHeader(302)
		return
	}
	//teams
	if req.URL.String() == "/getTeamsDeviceUsageDistributionUserCounts(period='D7')" {
		if err := checkMethodAndAuth(rw, req); err != nil {
			return
		}
		rw.Header().Add("Content-Type", "text/plain")
		rw.Header().Add("Location", "/TeamsDeviceUsageDistributionUserCountsCSV")
		rw.WriteHeader(302)
		return
	}
	if req.URL.String() == "/getTeamsUserActivityCounts(period='D7')" {
		if err := checkMethodAndAuth(rw, req); err != nil {
			return
		}
		rw.Header().Add("Content-Type", "text/plain")
		rw.Header().Add("Location", "/TeamsUserActivityCountsCSV")
		rw.WriteHeader(302)
		return
	}
	//onedrive
	if req.URL.String() == "/getOneDriveUsageFileCounts(period='D7')" {
		if err := checkMethodAndAuth(rw, req); err != nil {
			return
		}
		rw.Header().Add("Content-Type", "text/plain")
		rw.Header().Add("Location", "/OneDriveUsageFileCountsCSV")
		rw.WriteHeader(302)
		return
	}
	if req.URL.String() == "/getOneDriveActivityUserCounts(period='D7')" {
		if err := checkMethodAndAuth(rw, req); err != nil {
			return
		}
		rw.Header().Add("Content-Type", "text/plain")
		rw.Header().Add("Location", "/OneDriveActivityUserCountsCSV")
		rw.WriteHeader(302)
		return
	}
	//Outlook
	if req.URL.String() == "/getMailboxUsageMailboxCounts(period='D7')" {
		if err := checkMethodAndAuth(rw, req); err != nil {
			return
		}
		rw.Header().Add("Content-Type", "text/plain")
		rw.Header().Add("Location", "/MailboxUsageMailboxCountsCSV")
		rw.WriteHeader(302)
		return
	}
	if req.URL.String() == "/getEmailActivityCounts(period='D7')" {
		if err := checkMethodAndAuth(rw, req); err != nil {
			return
		}
		rw.Header().Add("Content-Type", "text/plain")
		rw.Header().Add("Location", "/EmailActivityCountsCSV")
		rw.WriteHeader(302)
		return
	}
	if req.URL.String() == "/getMailboxUsageStorage(period='D7')" {
		if err := checkMethodAndAuth(rw, req); err != nil {
			return
		}
		rw.Header().Add("Content-Type", "text/plain")
		rw.Header().Add("Location", "/MailboxUsageStorageCSV")
		rw.WriteHeader(302)
		return
	}
	if req.URL.String() == "/getEmailAppUsageAppsUserCounts(period='D7')" {
		if err := checkMethodAndAuth(rw, req); err != nil {
			return
		}
		rw.Header().Add("Content-Type", "text/plain")
		rw.Header().Add("Location", "/EmailAppUsageAppsUserCountsCSV")
		rw.WriteHeader(302)
		return
	}
	if req.URL.String() == "/getMailboxUsageQuotaStatusMailboxCounts(period='D7')" {
		if err := checkMethodAndAuth(rw, req); err != nil {
			return
		}
		rw.Header().Add("Content-Type", "text/plain")
		rw.Header().Add("Location", "/MailboxUsageQuotaStatusMailboxCountsCSV")
		rw.WriteHeader(302)
		return
	}

	////////////		Redirects
	//sharepoint
	if req.URL.String() == "/SharePointSiteUsageFileCountsCSV" {
		rw.WriteHeader(200)
		rw.Header().Add("Content-Type", "application/octet-stream")
		rw.Write([]byte(
			"Report Refresh Date,Site Type,Total,Active,Report Date,Report Period\n2023-04-25,All,2,0,2023-04-25,7\n2023-04-25,All,2,0,2023-04-24,7\n2023-04-25,All,2,0,2023-04-23,7\n2023-04-25,All,2,0,2023-04-22,7\n2023-04-25,All,2,0,2023-04-21,7\n2023-04-25,All,2,0,2023-04-20,7\n2023-04-25,All,2,0,2023-04-19,7\n",
		))
		return
	}
	if req.URL.String() == "/SharePointSiteUsageSiteCountsCSV" {
		rw.WriteHeader(200)
		rw.Header().Add("Content-Type", "application/octet-stream")
		rw.Write([]byte(
			"Report Refresh Date,Site Type,Total,Active,Report Date,Report Period\n22023-04-25,All,14,3,2023-04-25,7\n2023-04-24,All,11,8,2023-04-25,7\n2023-04-23,All,12,2,2023-04-25,7\n2023-04-22,All,18,6,2023-04-25,7\n2023-04-21,All,15,9,2023-04-25,7\n2023-04-20,All,17,1,2023-04-25,7\n2023-04-19,All,19,4,2023-04-25,7\n",
		))
		return
	}
	if req.URL.String() == "/SharePointSiteUsagePagesCSV" {
		rw.WriteHeader(200)
		rw.Header().Add("Content-Type", "application/octet-stream")
		rw.Write([]byte(
			"Report Refresh Date,Site Type,Page View Count,Report Date,Report Period\n2023-04-25,All,5,2023-04-25,7\n2023-04-24,All,3,2023-04-25,7\n2023-04-23,All,0,2023-04-25,7\n2023-04-22,All,8,2023-04-25,7\n2023-04-21,All,7,2023-04-25,7\n2023-04-20,All,4,2023-04-25,7\n2023-04-19,All,9,2023-04-25,7\n",
		))
		return
	}
	if req.URL.String() == "/SharePointActivityPagesCSV" {
		rw.WriteHeader(200)
		rw.Header().Add("Content-Type", "application/octet-stream")
		rw.Write([]byte(
			"Report Refresh Date,Visited Page Count,Report Date,Report Period\n2023-04-25,3,2023-04-25,7\n2023-04-24,6,2023-04-25,7\n2023-04-23,8,2023-04-25,7\n2023-04-22,2,2023-04-25,7\n2023-04-21,9,2023-04-25,7\n2023-04-20,1,2023-04-25,7\n2023-04-19,7,2023-04-25,7\n",
		))
		return
	}
	if req.URL.String() == "/SharePointSiteUsageStorageCSV" {
		rw.WriteHeader(200)
		rw.Header().Add("Content-Type", "application/octet-stream")
		rw.Write([]byte(
			"Report Refresh Date,Site Type,Storage Used(Byte),Report Date,Report Period\n2023-04-25,All,1098,2023-04-25,7\n2023-04-24,All,971,2023-04-25,7\n2023-04-23,All,1683,2023-04-25,7\n2023-04-22,All,1322,2023-04-25,7\n2023-04-21,All,1218,2023-04-25,7\n2023-04-20,All,1179,2023-04-25,7\n2023-04-19,All,1873,2023-04-25,7\n",
		))
		return
	}
	//teams
	if req.URL.String() == "/TeamsDeviceUsageDistributionUserCountsCSV" {
		rw.WriteHeader(200)
		rw.Header().Add("Content-Type", "application/octet-stream")
		rw.Write([]byte(
			"Report Refresh Date,Web,Windows Phone,Android Phone,iOS,Mac,Windows,Chrome OS,Linux,Report Period\n2023-04-25,13,0,12,19,1,16,9,7,7\n",
		))
		return
	}
	if req.URL.String() == "/TeamsUserActivityCountsCSV" {
		rw.WriteHeader(200)
		rw.Header().Add("Content-Type", "application/octet-stream")
		rw.Write([]byte(
			"Report Refresh Date,Report Date,Team Chat Messages,Post Messages,Reply Messages,Private Chat Messages,Calls,Meetings,Audio Duration,Video Duration,Screen Share Duration,Meetings Organized,Meetings Attended,Report Period\n2023-04-25,2023-04-25,4,13,8,0,20,11,2,20,11,0,14,7\n2023-04-24,2023-04-25,11,8,9,9,13,3,2,2,16,8,7\n2023-04-23,2023-04-25,4,5,5,18,5,5,1,20,5,14,7\n2023-04-22,2023-04-25,18,3,3,11,20,1,1,8,1,5,7\n2023-04-21,2023-04-25,11,18,10,6,13,2,3,3,3,14,7\n2023-04-20,2023-04-25,19,6,18,17,11,17,4,4,4,4,7\n2023-04-19,2023-04-25,10,14,10,7,4,4,4,4,4,4,7\n",
		))
		return
	}
	//onedrive
	if req.URL.String() == "/OneDriveUsageFileCountsCSV" {
		rw.WriteHeader(200)
		rw.Header().Add("Content-Type", "application/octet-stream")
		rw.Write([]byte(
			"Report Refresh Date,Site Type,Total,Active,Report Date,Report Period\n2023-04-25,All,20,6,2023-04-25,7\n2023-04-24,All,19,2,2023-04-25,7\n2023-04-23,All,19,6,2023-04-25,7\n2023-04-22,All,18,3,2023-04-25,7\n2023-04-21,All,10,8,2023-04-25,7\n2023-04-20,All,18,6,2023-04-25,7\n2023-04-19,All,10,4,2023-04-25,7\n",
		))
		return
	}
	if req.URL.String() == "/OneDriveActivityUserCountsCSV" {
		rw.WriteHeader(200)
		rw.Header().Add("Content-Type", "application/octet-stream")
		rw.Write([]byte(
			"Report Refresh Date,Viewed Or Edited,Synced,Shared Internally,Shared Externally,Report Date,Report Period\n2023-04-25,16,4,4,20,2023-04-25,7\n2023-04-24,0,16,13,10,2023-04-25,7\n2023-04-23,10,10,0,9,2023-04-25,7\n2023-04-22,20,13,16,2,2023-04-25,7\n2023-04-21,19,8,19,2,2023-04-25,7\n2023-04-20,1,7,1,3,2023-04-25,7\n2023-04-19,5,6,5,6,2023-04-25,7\n",
		))
		return
	}
	//outlook
	if req.URL.String() == "/MailboxUsageMailboxCountsCSV" {
		rw.WriteHeader(200)
		rw.Header().Add("Content-Type", "application/octet-stream")
		rw.Write([]byte(
			"Report Refresh Date,Total,Active,Report Date,Report Period\n2023-04-25,12,3,2023-04-25,7\n2023-04-24,17,6,2023-04-25,7\n2023-04-23,14,5,2023-04-25,7\n2023-04-22,18,2,2023-04-25,7\n2023-04-21,10,8,2023-04-25,7\n2023-04-20,19,1,2023-04-25,7\n2023-04-19,13,0,2023-04-25,7\n",
		))
		return
	}
	if req.URL.String() == "/EmailActivityCountsCSV" {
		rw.WriteHeader(200)
		rw.Header().Add("Content-Type", "application/octet-stream")
		rw.Write([]byte(
			"Report Refresh Date,Send,Receive,Read,Meeting Created,Meeting Interacted,Report Date,Report Period\n2023-04-25,6,1,7,0,6,2023-04-25,7\n2023-04-24,7,1,15,5,1,2023-04-25,7\n2023-04-23,7,3,3,3,18,2023-04-25,7\n2023-04-22,18,10,2,12,8,2023-04-25,7\n2023-04-21,9,7,13,10,4,2023-04-25,7\n2023-04-20,3,8,0,10,20,2023-04-25,7\n2023-04-19,15,3,18,13,3,2023-04-25,7\n",
		))
		return
	}
	if req.URL.String() == "/MailboxUsageStorageCSV" {
		rw.WriteHeader(200)
		rw.Header().Add("Content-Type", "application/octet-stream")
		rw.Write([]byte(
			"Report Refresh Date,Storage Used (Byte),Report Date,Report Period\n2023-04-25,1635,2023-04-25,7\n2023-04-24,1222,2023-04-25,7\n2023-04-23,967,2023-04-25,7\n2023-04-22,1969,2023-04-25,7\n2023-04-21,567,2023-04-25,7\n2023-04-20,707,2023-04-25,7\n2023-04-19,1423,2023-04-25,7\n",
		))
		return
	}
	if req.URL.String() == "/EmailAppUsageAppsUserCountsCSV" {
		rw.WriteHeader(200)
		rw.Header().Add("Content-Type", "application/octet-stream")
		rw.Write([]byte(
			"Report Refresh Date,Mail For Mac,Outlook For Mac,Outlook For Windows,Outlook For Mobile,Other For Mobile,Outlook For Web,POP3 App,IMAP4 App,SMTP App,Report Period\n2023-04-25,10,5,17,1,11,10,2,8,7,7\n2023-04-24,0,20,0,10,14,11,13,1,12,7\n2023-04-23,9,9,4,4,8,7,17,18,6,7\n2023-04-22,20,0,13,10,9,11,18,3,3,7\n2023-04-21,3,8,7,6,14,12,9,2,2,7\n2023-04-20,11,7,14,10,20,19,19,18,9,7\n2023-04-19,2,3,7,12,16,18,7,8,17,7\n",
		))
		return
	}
	if req.URL.String() == "/MailboxUsageQuotaStatusMailboxCountsCSV" {
		rw.WriteHeader(200)
		rw.Header().Add("Content-Type", "application/octet-stream")
		rw.Write([]byte(
			"Report Refresh Date,Under Limit,Warning Issued,Send Prohibited,Send/Receive Prohibited,Indeterminate,Report Date,Report Period\n2023-04-25,0,6,20,6,11,2023-04-25,7\n2023-04-24,1,0,1,2,2,2023-04-25,7\n2023-04-23,7,9,10,0,7,2023-04-25,7\n2023-04-22,5,5,10,10,19,2023-04-25,7\n2023-04-21,5,5,20,18,12,2023-04-25,7\n2023-04-20,3,12,1,8,2,2023-04-25,7\n2023-04-19,19,20,14,19,7,2023-04-25,7\n",
		))
		return
	}
	rw.WriteHeader(404)
	return

}
