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

// Package m365receiver import "github.com/observiq/bindplane-agent/receiver/m365receiver"
package m365receiver

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type m365Client struct {
	client       *http.Client
	authEndpoint string
	clientID     string
	clientSecret string
	token        string
	scope        string
}

// Option func for GetJSON request handler case
type Option func(r *http.Request)

// api error struct
type apiError struct {
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Message string `json:"Message,omitempty"`
}

// return type for logs
type logData struct {
	log  jsonLog
	body string
}

// JSON parsing structs
type jsonLog struct {
	// common schema attributes
	UserType     int    `json:"UserType"`
	RecordType   int    `json:"RecordType"`
	UserID       string `json:"UserId"`
	CreationTime string `json:"CreationTime"`
	ID           string `json:"Id"`
	Operation    string `json:"Operation"`
	Workload     string `json:"Workload,omitempty"`
	ResultStatus string `json:"ResultStatus,omitempty"`

	// optional schema attributes
	SharepointSite           string              `json:"Site,omitempty"`
	SharepointSourceFileName string              `json:"SourceFileName,omitempty"`
	ExchangeMailboxGUID      string              `json:"MailboxGuid,omitempty"`
	SecurityAlertID          string              `json:"AlertId,omitempty"`
	SecurityAlertName        string              `json:"Name,omitempty"`
	YammerActorID            string              `json:"ActorUserId,omitempty"`
	DefenderURL              string              `json:"URL,omitempty"`
	InvestigationID          string              `json:"InvestigationId,omitempty"`
	InvestigationStatus      string              `json:"Status,omitempty"`
	PowerAppName             string              `json:"AppName,omitempty"`
	DynamicsEntityID         string              `json:"EntityId,omitempty"`
	DynamicsEntityName       string              `json:"EntityName,omitempty"`
	FormID                   string              `json:"FormId,omitempty"`
	MIPLabelID               string              `json:"LabelId,omitempty"`
	EncryptedMessageID       string              `json:"MessageId,omitempty"`
	ConnectorJobID           string              `json:"JobId,omitempty"`
	ConnectorTaskID          string              `json:"TaskId,omitempty"`
	MSGraphConsentAppID      string              `json:"ApplicationId,omitempty"`
	VivaGoalsUsername        string              `json:"Username,omitempty"`
	VivaGoalsOrgName         string              `json:"OrganizationName,omitempty"`
	MSToDoAppID              string              `json:"ActorAppId,omitempty"`
	MSToDoItemID             string              `json:"ItemID,omitempty"`
	MSWebProjectID           string              `json:"ProjectId,omitempty"`
	MSWebRoadmapID           string              `json:"RoadmapId,omitempty"`
	MSWebRoadmapItemID       string              `json:"RoadmapItemId,omitempty"`
	DefenderFileSource       *int                `json:"SourceWorkload,omitempty"`
	QuarantineSource         *int                `json:"RequestSource,omitempty"`
	YammerFileID             *int                `json:"FileId,omitempty"`
	CommCompliance           *ExchangeDetails    `json:"ExchangeDetails,omitempty"`
	DataShareInvitation      *Invitation         `json:"Invitation,omitempty"`
	DefenderFile             *FileData           `json:"FileData,omitempty"`
	DLPSharePointMetaData    *SharePointMetaData `json:"SharePointMetaData,omitempty"`
	DLPExchangeMetaData      *ExchangeMetaData   `json:"ExchangeMetaData,omitempty"`
	DefenderEmail            *[]AttachmentData   `json:"AttachmentData,omitempty"`
	AzureActor               *[]AzureActor       `json:"Actor,omitempty"`
	DLPPolicyDetails         *[]PolicyDetails    `json:"PolicyDetails,omitempty"`
}

// AzureActor json struct
type AzureActor struct {
	ID   string `json:"ID"`
	Type int    `json:"Type"`
}

// SharePointMetaData json struct
type SharePointMetaData struct {
	From string `json:"From"`
}

// ExchangeMetaData json struct
type ExchangeMetaData struct {
	MessageID string `json:"MessageID"`
}

// PolicyDetails json struct
type PolicyDetails struct {
	PolicyID   string `json:"PolicyId"`
	PolicyName string `json:"PolicyName"`
}

// AttachmentData json struct
type AttachmentData struct {
	FileName string `json:"FileName"`
}

// FileData json struct
type FileData struct {
	DocumentID  string `json:"DocumentId"`
	FileVerdict int    `json:"FileVerdict"`
}

// ExchangeDetails json struct
type ExchangeDetails struct {
	NetworkMessageID string `json:"NetworkMessageId,omitempty"`
}

// Invitation json struct
type Invitation struct {
	ShareID string `json:"ShareId"`
}

// client implementation
func newM365Client(c *http.Client, cfg *Config, scope string) *m365Client {
	return &m365Client{
		client:       c,
		authEndpoint: fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", cfg.TenantID),
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		scope:        scope,
	}
}

func (m *m365Client) shutdown() error {
	m.client.CloseIdleConnections()
	return nil
}

// Get authorization token
// metrics token has scope = "https://graph.microsoft.com/.default"
// logs token has scope = "https://manage.office.com/.default"
func (m *m365Client) GetToken(ctx context.Context) error {
	formData := url.Values{
		"grant_type":    {"client_credentials"},
		"scope":         {m.scope},
		"client_id":     {m.clientID},
		"client_secret": {m.clientSecret},
	}
	requestBody := strings.NewReader(formData.Encode())
	resp, err := m.makeRequest(ctx, requestBody, "POST", m.authEndpoint, WithBody())
	if err != nil {
		return err
	}

	defer func() { _ = resp.Body.Close() }()

	// troubleshoot resp err if present
	if resp.StatusCode != 200 {
		// get error code
		respErr := struct {
			Err string `json:"error"`
		}{}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		err = json.Unmarshal(body, &respErr)
		if err != nil {
			return err
		}
		//match on error code
		switch respErr.Err {
		case "unauthorized_client":
			return fmt.Errorf("the provided client_id is incorrect or does not exist within the given tenant directory")
		case "invalid_client":
			return fmt.Errorf("the provided client_secret is incorrect or does not belong to the given client_id")
		case "invalid_request":
			return fmt.Errorf("the provided tenant_id is incorrect or does not exist")
		}
		return fmt.Errorf("got non 200 status code from request, got %d", resp.StatusCode)
	}

	// read in access token
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	token := struct {
		Token string `json:"access_token"`
	}{}
	err = json.Unmarshal(body, &token)
	if err != nil {
		return err
	}
	m.token = token.Token
	return nil
}

// function for getting metrics data
func (m *m365Client) GetCSV(ctx context.Context, endpoint string) ([]string, error) {
	resp, err := m.makeRequest(ctx, nil, "GET", endpoint, WithToken(m.token))
	if err != nil {
		return []string{}, err
	}

	//troubleshoot resp err if present
	if resp.StatusCode != 200 {
		return []string{}, m.handleErrors(resp)
	}

	defer func() { _ = resp.Body.Close() }()
	csvReader := csv.NewReader(resp.Body)

	//parse out 2nd line & return csv data
	_, err = csvReader.Read()
	if err != nil {
		return []string{}, err
	}
	data, err := csvReader.Read()
	if err != nil {
		if err == io.EOF { //no data in report, scraper should still run
			return []string{}, nil
		}
		return []string{}, err
	}

	return data, nil
}

// function for getting log data
func (m *m365Client) GetJSON(ctx context.Context, endpoint string, end string, start string) ([]logData, error) {
	// make request to Audit endpoint
	resp, err := m.makeRequest(ctx, nil, "GET", endpoint, WithToken(m.token), WithTime(end, start))
	if err != nil {
		return []logData{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	// troubleshoot error code
	if resp.StatusCode != 200 {
		return []logData{}, m.handleErrors(resp)
	}

	// get redirect links
	redirectLinks, err := m.readInContent(resp)
	if err != nil {
		return []logData{}, err
	}
	if redirectLinks == nil { // if no data, return no err
		return []logData{}, nil
	}

	// follows redirect link(s) to get json array of logs
	// parses top level array objects
	var data = []logData{}
	for _, l := range redirectLinks {
		body, err := m.followLink(ctx, l)
		if err != nil {
			return []logData{}, err
		}
		bodies, err := m.parseBody(body)
		if err != nil {
			return []logData{}, err
		}
		for _, b := range bodies {
			var log = jsonLog{}
			err = json.Unmarshal(b, &log)
			data = append(data, logData{log: log, body: string(b)})
		}
	}

	return data, nil
}

func (m *m365Client) StartSubscription(ctx context.Context, endpoint string) error {
	resp, err := m.makeRequest(ctx, nil, "POST", endpoint, WithToken(m.token))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// troubleshoot error handling, mainly sub already enabled
	// no error if sub already enabled, not troubleshooting stale token
	// only called while code is synchronous right after a GetToken call
	// if token is stale, regenerating it won't fix anything
	if resp.StatusCode != 200 {
		return m.handleErrors(resp)
	}
	//if StatusCode == 200, then subscription was successfully started
	return nil
}

// followLink will follow the response of a first request that has a link to the actual content
func (m *m365Client) followLink(ctx context.Context, endpoint string) ([]byte, error) {
	resp, err := m.makeRequest(ctx, nil, "GET", endpoint, WithToken(m.token))
	if err != nil {
		return []byte{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	// troubleshoot error code
	if resp.StatusCode != 200 {
		return []byte{}, m.handleErrors(resp)
	}

	return io.ReadAll(resp.Body)
}

// handles error reading for client funcs
func (m *m365Client) handleErrors(r *http.Response) error {
	respErr, err := m.readInErr(r)
	if err != nil {
		return err
	}
	if respErr.Message != "" && respErr.Message == "Authorization has been denied for this request." { // bad token
		return fmt.Errorf("authorization denied")
	}
	if respErr.Error != nil {
		if respErr.Error.Code == "AF20024" { // sub already started, not an issue
			return nil
		} else if respErr.Error.Code == "InvalidAuthenticationToken" { // stale token error, potentially
			return fmt.Errorf("access token invalid")
		}
	}
	return fmt.Errorf("got non 200 status code from request, got %d", r.StatusCode)
}

// parseBody takes the byte[] response and parses it into string objects
// restores strings to json format after splits and trims
func (m *m365Client) parseBody(body []byte) ([]json.RawMessage, error) {
	var rawMessages []json.RawMessage
	err := json.Unmarshal(body, &rawMessages)
	if err != nil {
		return []json.RawMessage{}, err
	}
	return rawMessages, nil
}

// reads in errors for responses
func (m *m365Client) readInErr(resp *http.Response) (apiError, error) {
	var respErr apiError
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return apiError{}, err
	}
	err = json.Unmarshal(body, &respErr)
	if err != nil {
		return apiError{}, err
	}
	return respErr, nil
}

// reads in content uri for GetJSON
func (m *m365Client) readInContent(resp *http.Response) ([]string, error) {
	// check if contentUri field if present
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []string{}, err
	}
	if string(body) == "[]" { // if body is empty, no new log data available
		return []string{}, nil
	}
	// extract contentUri field
	lr := []struct {
		Content string `json:"contentUri"`
	}{}
	if err = json.Unmarshal(body, &lr); err != nil {
		return []string{}, err
	}
	var endpoints = []string{}
	for _, i := range lr {
		endpoints = append(endpoints, i.Content)
	}

	return endpoints, nil
}

// handles requests for client functions
func (m *m365Client) makeRequest(ctx context.Context, body io.Reader, httpMethod, endpoint string, opts ...Option) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, httpMethod, endpoint, body)
	if err != nil {
		return nil, err
	}
	for _, o := range opts {
		o(req)
	}

	return m.client.Do(req)
}

// WithBody is an option func for the request handler
func WithBody() Option {
	return func(r *http.Request) {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
}

// WithToken is an option func for the request handler
func WithToken(token string) Option {
	return func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer "+token)
	}
}

// WithTime is an option func for the request handler
func WithTime(end, start string) Option {
	return func(r *http.Request) {
		q := r.URL.Query()
		q.Add("startTime", start)
		q.Add("endTime", end)
		r.URL.RawQuery = q.Encode()
	}
}
