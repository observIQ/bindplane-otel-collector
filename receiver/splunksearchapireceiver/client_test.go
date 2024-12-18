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

package splunksearchapireceiver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateSearchJob(t *testing.T) {
	server := newMockServer()
	testClient := defaultSplunkSearchAPIClient{
		client:   server.Client(),
		endpoint: server.URL,
	}

	resp, err := testClient.CreateSearchJob("index=otel starttime=\"\" endtime=\"\" timeformat=\"\"")
	require.NoError(t, err)
	require.Equal(t, "123456", resp.SID)

	// returns an error if the search doesn't have times
	resp, err = testClient.CreateSearchJob("index=otel")
	require.EqualError(t, err, "search query must contain starttime, endtime, and timeformat")
	require.Empty(t, resp)

	// returns an error if the response status isn't 201
	resp, err = testClient.CreateSearchJob("index=fail_to_create_job starttime=\"\" endtime=\"\" timeformat=\"\"")
	require.ErrorContains(t, err, "create search job")
	require.Empty(t, resp)

	// returns an error if the response body can't be unmarshalled
	resp, err = testClient.CreateSearchJob("index=fail_to_unmarshal starttime=\"\" endtime=\"\" timeformat=\"\"")
	require.ErrorContains(t, err, "unmarshal search job create response")
	require.Empty(t, resp)

}

func TestGetJobStatus(t *testing.T) {
	server := newMockServer()
	testClient := defaultSplunkSearchAPIClient{
		client:   server.Client(),
		endpoint: server.URL,
	}

	resp, err := testClient.GetJobStatus("123456")
	require.NoError(t, err)
	require.Equal(t, "DONE", resp.Content.Dict.Keys[0].Value)
	require.Equal(t, "text/xml", resp.Content.Type)

	// returns an error if the response status isn't 200
	resp, err = testClient.GetJobStatus("654321")
	require.ErrorContains(t, err, "get search job status")
	require.Empty(t, resp)

	// returns an error if the response body can't be unmarshalled
	resp, err = testClient.GetJobStatus("098765")
	require.ErrorContains(t, err, "unmarshal search job status response")
	require.Empty(t, resp)
}

func TestGetSearchResults(t *testing.T) {
	server := newMockServer()
	testClient := defaultSplunkSearchAPIClient{
		client:   server.Client(),
		endpoint: server.URL,
	}

	resp, err := testClient.GetSearchResults("123456", 0, 5)
	require.NoError(t, err)
	require.Equal(t, 3, len(resp.Results))
	require.Equal(t, "Hello, world!", resp.Results[0].Raw)

	// returns an error if the response status isn't 200
	resp, err = testClient.GetSearchResults("654321", 0, 5)
	require.ErrorContains(t, err, "get search job results")
	require.Empty(t, resp)

	// returns an error if the response body can't be unmarshalled
	resp, err = testClient.GetSearchResults("098765", 0, 5)
	require.ErrorContains(t, err, "unmarshal search job results response")
	require.Empty(t, resp)
}

func TestSetSplunkRequestAuth(t *testing.T) {
	client := defaultSplunkSearchAPIClient{
		username: "user",
		password: "password",
	}
	req := httptest.NewRequest("GET", "http://localhost:8089", nil)
	client.setSplunkRequestAuth(req)
	require.Equal(t, req.Header.Get("Authorization"), "Basic dXNlcjpwYXNzd29yZA==")

	client = defaultSplunkSearchAPIClient{
		authToken: "token",
		tokenType: TokenTypeBearer,
	}
	client.setSplunkRequestAuth(req)
	require.Equal(t, req.Header.Get("Authorization"), "Bearer token")

	client = defaultSplunkSearchAPIClient{
		authToken: "token",
		tokenType: TokenTypeSplunk,
	}
	client.setSplunkRequestAuth(req)
	require.Equal(t, req.Header.Get("Authorization"), "Splunk token")
}

// mock Splunk servers
func newMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		switch req.URL.String() {
		case "/services/search/jobs":
			body, _ := io.ReadAll(req.Body)
			if strings.Contains(string(body), "search=index%3Dotel") {
				rw.Header().Set("Content-Type", "application/xml")
				rw.WriteHeader(http.StatusCreated)
				rw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
				<response>
					<sid>123456</sid>
				</response>
				`))
			}
			if strings.Contains(string(body), "index%3Dfail_to_create_job") {
				rw.WriteHeader(http.StatusNotFound)
			}
			if strings.Contains(string(body), "index%3Dfail_to_unmarshal") {
				rw.WriteHeader(http.StatusCreated)
				rw.Write([]byte(`invalid xml`))
			}
		case "/services/search/v2/jobs/123456":
			rw.Header().Set("Content-Type", "application/xml")
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
			<response>
				<content type="text/xml">
					<dict>
						<key name="dispatchState">DONE</key>
					</dict>
				</content>
			</response>`))
		case "/services/search/v2/jobs/654321":
			rw.WriteHeader(http.StatusNotFound)
		case "/services/search/v2/jobs/098765":
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(`invalid xml`))
		case "/services/search/v2/jobs/123456/results?output_mode=json&offset=0&count=5":
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(http.StatusOK)
			rw.Write(splunkEventsResultsP1)
		case "/services/search/v2/jobs/654321/results?output_mode=json&offset=0&count=5":
			rw.WriteHeader(http.StatusNotFound)
		case "/services/search/v2/jobs/098765/results?output_mode=json&offset=0&count=5":
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(`invalid json`))
		default:
			rw.WriteHeader(http.StatusNotFound)
		}
	}))
}
