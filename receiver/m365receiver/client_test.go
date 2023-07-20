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

package m365receiver //import "github.com/observiq/bindplane-agent/receiver/m365receiver"

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetToken(t *testing.T) {
	m365Mock := newMockServerToken()
	testClient := newM365Client(m365Mock.Client(), &Config{}, "https://graph.microsoft.com/.default")
	testClient.authEndpoint = m365Mock.URL + "/testTenantID"
	testClient.clientID = "testClientID"
	testClient.clientSecret = "testClientSecret"

	// test 1: correct behavior
	err := testClient.GetToken(context.Background())
	require.NoError(t, err)
	require.Equal(t, "testAccessToken", testClient.token)

	// test 2: incorrect client secret
	testClient.clientSecret = "err"
	err = testClient.GetToken(context.Background())
	assert.EqualError(t, err, "the provided client_secret is incorrect or does not belong to the given client_id")

	// test 3: incorrect client id
	testClient.clientSecret = "testClientSecret"
	testClient.clientID = "err"
	err = testClient.GetToken(context.Background())
	assert.EqualError(t, err, "the provided client_id is incorrect or does not exist within the given tenant directory")

	// test 4: incorrect tenant_id
	testClient.clientID = "testClientID"
	testClient.authEndpoint = m365Mock.URL + "/err"
	err = testClient.GetToken(context.Background())
	assert.EqualError(t, err, "the provided tenant_id is incorrect or does not exist")
}

func TestGetCSV(t *testing.T) {
	m365Mock := newMockServerCSV()
	testClient := newM365Client(m365Mock.Client(), &Config{}, "https://graph.microsoft.com/.default")
	testClient.token = "foo"

	//expected behavior
	testLine, err := testClient.GetCSV(context.Background(), m365Mock.URL+"/getSharePointSiteUsageFileCounts(period='D7')")
	require.NoError(t, err)
	expectedLine := []string{"2023-04-25", "All", "2", "0", "2023-04-25", "7"}
	require.Equal(t, expectedLine, testLine)

	//test no returned data
	testLine, err = testClient.GetCSV(context.Background(), m365Mock.URL+"/testNoData")
	require.NoError(t, err)
	expectedLine = []string{}
	require.Equal(t, expectedLine, testLine)

	//err testing
	testClient.token = "err"
	_, err = testClient.GetCSV(context.Background(), m365Mock.URL+"/getSharePointSiteUsageFileCounts(period='D7')")
	assert.EqualError(t, err, "access token invalid")
}

func TestStartSubscription(t *testing.T) {
	m365Mock := newMockServerSub()
	testClient := newM365Client(m365Mock.Client(), &Config{}, "https://manage.office.com/.default")
	testClient.token = "foo"

	//expected behavior
	err := testClient.StartSubscription(context.Background(), m365Mock.URL+"/testStartSub")
	require.NoError(t, err)

	//sub already enabled
	err = testClient.StartSubscription(context.Background(), m365Mock.URL+"/testSubStartedAlready")
	require.NoError(t, err)
}

func TestGetJSON(t *testing.T) {
	m365Mock := newMockServerJSON()
	testClient := newM365Client(m365Mock.Client(), &Config{}, "https://manage.office.com/.default")
	testClient.token = "foo"

	//expected behavior
	testJSON, err := testClient.GetJSON(context.Background(), m365Mock.URL+"/testJSON", "", "")
	require.NoError(t, err)
	expectedJSON := logData{
		log: jsonLog{
			Workload:     "testWorkload",
			UserID:       "testUserId",
			RecordType:   0,
			UserType:     0,
			CreationTime: "2023-05-09T22:25:14",
			ID:           "testId",
			Operation:    "testOperation",
			ResultStatus: "testResultStatus",
		},
		body: "{\"CreationTime\": \"2023-05-09T22:25:14\",\"Id\": \"testId\",\"Operation\": \"testOperation\",\"OrganizationID\": \"testOrgId\",\"ResultStatus\": \"testResultStatus\",\"UserId\": \"testUserId\",\"UserType\": 0,\"Workload\": \"testWorkload\"}",
	}
	require.Equal(t, expectedJSON, testJSON[0])

	// bad token
	testClient.token = "bad"
	testJSON, err = testClient.GetJSON(context.Background(), m365Mock.URL+"/testJSON", "", "")
	require.EqualError(t, err, "authorization denied")
}

func TestFollowLinkErr(t *testing.T) {
	m365Mock := newMockServerJSON()
	testClient := newM365Client(m365Mock.Client(), &Config{}, "https://manage.office.com/.default")
	testClient.token = "bad"

	_, err := testClient.followLink(context.Background(), m365Mock.URL+"/testJSONredirect")
	require.EqualError(t, err, "authorization denied")
}

//	Mock Servers

func newMockServerCSV() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.String() == "/getSharePointSiteUsageFileCounts(period='D7')" {
			if req.Method != "GET" {
				rw.WriteHeader(400)
				rw.Write([]byte("Error, incorrect HTTP method"))
				return
			}

			if a := req.Header.Get("Authorization"); a != "Bearer foo" {
				rw.WriteHeader(400)
				rw.Write([]byte(`{"error": {"code": "InvalidAuthenticationToken"}}`))
				return
			}

			rw.WriteHeader(200)
			rw.Write([]byte(
				"Report Refresh Date,Site Type,Total,Active,Report Date,Report Period\n2023-04-25,All,2,0,2023-04-25,7\n2023-04-25,All,2,0,2023-04-24,7\n2023-04-25,All,2,0,2023-04-23,7\n2023-04-25,All,2,0,2023-04-22,7\n2023-04-25,All,2,0,2023-04-21,7\n2023-04-25,All,2,0,2023-04-20,7\n2023-04-25,All,2,0,2023-04-19,7\n",
			))
			return
		}
		if req.URL.String() == "/testNoData" {
			if req.Method != "GET" {
				rw.WriteHeader(400)
				rw.Write([]byte("Error, incorrect HTTP method"))
				return
			}

			if a := req.Header.Get("Authorization"); a != "Bearer foo" {
				rw.WriteHeader(400)
				rw.Write([]byte(`{"error": {"code": "InvalidAuthenticationToken"}}`))
				return
			}

			rw.WriteHeader(200)
			rw.Write([]byte(
				"Report Refresh Date,Site Type,Total,Active,Report Date,Report Period\n",
			))
			return
		}
		rw.WriteHeader(404)
		return
	}))
}

func newMockServerJSON() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/testJSON" {
			if req.Method != "GET" {
				rw.WriteHeader(405)
				rw.Write([]byte("error, incorrect HTTP method"))
				return
			}

			if a := req.Header.Get("Authorization"); a != "Bearer foo" {
				rw.WriteHeader(401)
				rw.Write([]byte(`{"Message": "Authorization has been denied for this request."}`))
				return
			}

			rw.WriteHeader(200)
			rw.Write([]byte(fmt.Sprintf(
				`[{
					"contentUri": "%s/testJSONredirect"
				}]`, "http://"+req.Host)))
			return

		}
		if req.URL.String() == "/testJSONredirect" {
			if req.Method != "GET" {
				rw.WriteHeader(405)
				rw.Write([]byte("error, incorrect HTTP method"))
				return
			}

			if a := req.Header.Get("Authorization"); a != "Bearer foo" {
				rw.WriteHeader(401)
				rw.Write([]byte(`{"Message": "Authorization has been denied for this request."}`))
				return
			}

			rw.WriteHeader(200)
			rw.Write([]byte(
				`[{"CreationTime": "2023-05-09T22:25:14","Id": "testId","Operation": "testOperation","OrganizationID": "testOrgId","ResultStatus": "testResultStatus","UserId": "testUserId","UserType": 0,"Workload": "testWorkload"}]`,
			))
		}
		rw.WriteHeader(404)
	}))
}

func newMockServerSub() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.String() == "/testStartSub" {
			if req.Method != "POST" {
				rw.WriteHeader(405)
				rw.Write([]byte("error, incorrect HTTP method"))
			}
			if a := req.Header.Get("Authorization"); a != "Bearer foo" {
				rw.WriteHeader(401)
				rw.Write([]byte(`{"Message": "Authorization has been denied for this request."}`))
				return
			}
			rw.WriteHeader(200)
		}
		if req.URL.String() == "/testSubStartedAlready" {
			if req.Method != "POST" {
				rw.WriteHeader(405)
				rw.Write([]byte("error, incorrect HTTP method"))
			}
			if a := req.Header.Get("Authorization"); a != "Bearer foo" {
				rw.WriteHeader(401)
				rw.Write([]byte(`{"Message": "Authorization has been denied for this request."}`))
				return
			}
			rw.WriteHeader(400)
			rw.Write([]byte(
				`{
					"error": {
						"code": "AF20024",
						"message": "The subscription is already enabled. No property change."
					}
				}`,
			))
		}
		rw.WriteHeader(404)
		return
	}))
}

func newMockServerToken() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.URL.String() == "/testTenantID" {
			if req.Method != "POST" {
				rw.WriteHeader(400)
				rw.Write([]byte("Error, incorrect HTTP method"))
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
				}`))
			return
		}
		if req.URL.String() == "/err" {
			rw.WriteHeader(400)
			rw.Write([]byte(`{"error": "invalid_request"}`))
		}
		rw.WriteHeader(404)
		return
	}))
}
