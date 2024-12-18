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

// Package splunksearchapireceiver contains the Splunk Search API receiver.
package splunksearchapireceiver

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type splunkSearchAPIClient interface {
	CreateSearchJob(search string) (CreateJobResponse, error)
	GetJobStatus(searchID string) (SearchJobStatusResponse, error)
	GetSearchResults(searchID string, offset int, batchSize int) (SearchResults, error)
}

type defaultSplunkSearchAPIClient struct {
	client    *http.Client
	endpoint  string
	logger    *zap.Logger
	username  string
	password  string
	authToken string
	tokenType string
}

func newDefaultSplunkSearchAPIClient(ctx context.Context, settings component.TelemetrySettings, conf Config, host component.Host) (*defaultSplunkSearchAPIClient, error) {
	client, err := conf.ClientConfig.ToClient(ctx, host, settings)
	if err != nil {
		return nil, err
	}

	return &defaultSplunkSearchAPIClient{
		client:    client,
		endpoint:  conf.Endpoint,
		logger:    settings.Logger,
		username:  conf.Username,
		password:  conf.Password,
		authToken: conf.AuthToken,
		tokenType: conf.TokenType,
	}, nil
}

func (c *defaultSplunkSearchAPIClient) CreateSearchJob(search string) (CreateJobResponse, error) {
	endpoint := fmt.Sprintf("%s/services/search/jobs", c.endpoint)

	if !strings.Contains(search, strings.ToLower("starttime=")) || !strings.Contains(search, strings.ToLower("endtime=")) || !strings.Contains(search, strings.ToLower("timeformat=")) {
		return CreateJobResponse{}, fmt.Errorf("search query must contain starttime, endtime, and timeformat")
	}

	reqBody := fmt.Sprintf(`search=%s`, url.QueryEscape(search))
	resp, err := c.doSplunkRequest("POST", endpoint, bytes.NewBuffer([]byte(reqBody)))
	if err != nil {
		return CreateJobResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return CreateJobResponse{}, fmt.Errorf("create search job: %d", resp.StatusCode)
	}

	var jobResponse CreateJobResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return CreateJobResponse{}, fmt.Errorf("read search job create response: %w", err)
	}

	err = xml.Unmarshal(body, &jobResponse)
	if err != nil {
		return CreateJobResponse{}, fmt.Errorf("unmarshal search job create response: %w", err)
	}
	return jobResponse, nil
}

func (c *defaultSplunkSearchAPIClient) GetJobStatus(sid string) (SearchJobStatusResponse, error) {
	endpoint := fmt.Sprintf("%s/services/search/v2/jobs/%s", c.endpoint, sid)

	resp, err := c.doSplunkRequest("GET", endpoint, nil)
	if err != nil {
		return SearchJobStatusResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return SearchJobStatusResponse{}, fmt.Errorf("get search job status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SearchJobStatusResponse{}, fmt.Errorf("read search job status response: %w", err)
	}
	var jobStatusResponse SearchJobStatusResponse
	err = xml.Unmarshal(body, &jobStatusResponse)
	if err != nil {
		return SearchJobStatusResponse{}, fmt.Errorf("unmarshal search job status response: %w", err)
	}

	return jobStatusResponse, nil
}

func (c *defaultSplunkSearchAPIClient) GetSearchResults(sid string, offset int, batchSize int) (SearchResults, error) {
	endpoint := fmt.Sprintf("%s/services/search/v2/jobs/%s/results?output_mode=json&offset=%d&count=%d", c.endpoint, sid, offset, batchSize)
	resp, err := c.doSplunkRequest("GET", endpoint, nil)
	if err != nil {
		return SearchResults{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return SearchResults{}, fmt.Errorf("get search job results: %d", resp.StatusCode)
	}

	var searchResults SearchResults
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SearchResults{}, fmt.Errorf("read search job results response: %w", err)
	}
	err = json.Unmarshal(body, &searchResults)
	if err != nil {
		return SearchResults{}, fmt.Errorf("unmarshal search job results response: %w", err)
	}

	return searchResults, nil
}

func (c *defaultSplunkSearchAPIClient) doSplunkRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("new http request: %w", err)
	}
	err = c.setSplunkRequestAuth(req)
	if err != nil {
		return nil, fmt.Errorf("set splunk request auth: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("client do request: %w", err)
	}
	return resp, nil
}

func (c *defaultSplunkSearchAPIClient) setSplunkRequestAuth(req *http.Request) error {
	if c.authToken != "" {
		if strings.EqualFold(c.tokenType, TokenTypeBearer) {
			req.Header.Set("Authorization", "Bearer "+string(c.authToken))
		} else if strings.EqualFold(c.tokenType, TokenTypeSplunk) {
			req.Header.Set("Authorization", "Splunk "+string(c.authToken))
		}
	} else {
		req.SetBasicAuth(c.username, c.password)
	}
	return nil
}
