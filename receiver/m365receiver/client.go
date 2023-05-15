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

// Package m365receiver import "github.com/observiq/observiq-otel-collector/receiver/m365receiver"
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

// HTTP parsing structs
type auth struct {
	Token string `json:"access_token"`
}
type tokenError struct {
	Err string `json:"error"`
}
type defaultError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type jsonError struct {
	Message string `json:"Message"`
}

type logResp struct {
	Content string `json:"contentUri"`
}

// JSON parsing structs
type jsonLogs struct {
	Workload                 string              `json:"Workload,omitempty"`
	UserId                   string              `json:"UserId"`
	UserType                 int                 `json:"UserType"`
	CreationTime             string              `json:"CreationTime"`
	Id                       string              `json:"Id"`
	Operation                string              `json:"Operation"`
	ResultStatus             string              `json:"ResultStatus,omitempty"`
	SharepointSite           string              `json:"Site,omitempty"`
	SharepointSourceFileName string              `json:"SourceFileName,omitempty"`
	ExchangeMailboxGUID      string              `json:"MailboxGuid,omitempty"`
	AzureActor               *AzureActor         `json:"Actor,omitempty"`
	DLPSharePointMetaData    *SharePointMetaData `json:"SharePointMetaData,omitempty"`
	DLPExchangeMetaData      *ExchangeMetaData   `json:"ExchangeMetaData,omitempty"`
	DLPPolicyDetails         *PolicyDetails      `json:"PolicyDetails,omitempty"`
	SecurityAlertId          string              `json:"AlertId,omitempty"`
	SecurityAlertName        string              `json:"Name,omitempty"`
	YammerActorId            string              `json:"ActorUserId,omitempty"`
	YammerFileId             *int                `json:"FileId,omitempty"`
	DefenderEmail            *AttachmentData     `json:"AttachmentData,omitempty"`
	DefenderURL              string              `json:"URL,omitempty"` //DefenderURLId            string             `json:"UserId,omitempty"`
	DefenderFile             *FileData           `json:"FileData,omitempty"`
	DefenderFileSource       *int                `json:"SourceWorkload,omitempty"`
	InvestigationId          string              `json:"InvestigationId,omitempty"`
	InvestigationStatus      string              `json:"Status,omitempty"`
	PowerAppName             string              `json:"AppName,omitempty"`
	DynamicsEntityId         string              `json:"EntityId,omitempty"`
	DynamicsEntityName       string              `json:"EntityName,omitempty"`
	QuarantineSource         *int                `json:"RequestSource,omitempty"`
	FormId                   string              `json:"FormId,omitempty"`
	MIPLabelId               string              `json:"LabelId,omitempty"`
	EncryptedMessageId       string              `json:"MessageId,omitempty"`
	CommCompliance           *ExchangeDetails    `json:"ExchangeDetails,omitempty"`
	ConnectorJobId           string              `json:"JobId,omitempty"`
	ConnectorTaskId          string              `json:"TaskId,omitempty"`
	DataShareInvitation      *Invitation         `json:"Invitation,omitempty"`
	MSGraphConsentAppId      string              `json:"ApplicationId,omitempty"`
	VivaGoalsUsername        string              `json:"Username,omitempty"`
	VivaGoalsOrgName         string              `json:"OrganizationName,omitempty"`
	MSToDoAppId              string              `json:"ActorAppId,omitempty"`
	MSToDoItemId             string              `json:"ItemID,omitempty"`
	MSWebProjectId           string              `json:"ProjectId,omitempty"`
	MSWebRoadmapId           string              `json:"RoadmapId,omitempty"`
	MSWebRoadmapItemId       string              `json:"RoadmapItemId,omitempty"`
}

type AzureActor struct {
	ID   string `json:"ID"`
	Type int    `json:"Type"`
}

type SharePointMetaData struct {
	From string `json:"From"`
}

type ExchangeMetaData struct {
	MessageID string `json:"MessageID"`
}

type PolicyDetails struct {
	PolicyId   string `json:"PolicyId"`
	PolicyName string `json:"PolicyName"`
}

type AttachmentData struct {
	FileName string `json:"FileName"`
}

type FileData struct {
	DocumentId  string `json:"DocumentId"`
	FileVerdict int    `json:"FileVerdict"`
}

type ExchangeDetails struct { //will this be nil if not present
	NetworkMessageId string `json:"NetworkMessageId,omitempty"`
}

type Invitation struct {
	ShareId string `json:"ShareId"`
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
func (m *m365Client) GetToken() error {
	formData := url.Values{
		"grant_type":    {"client_credentials"},
		"scope":         {m.scope},
		"client_id":     {m.clientID},
		"client_secret": {m.clientSecret},
	}

	requestBody := strings.NewReader(formData.Encode())

	req, err := http.NewRequest("POST", m.authEndpoint, requestBody)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}

	defer func() { _ = resp.Body.Close() }()

	//troubleshoot resp err if present
	if resp.StatusCode != 200 {
		//get error code
		var respErr tokenError
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var token auth
	err = json.Unmarshal(body, &token)
	if err != nil {
		return err
	}

	m.token = token.Token

	return nil
}

// function for getting metrics data
func (m *m365Client) GetCSV(endpoint string) ([]string, error) {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return []string{}, err
	}

	req.Header.Set("Authorization", m.token)
	resp, err := m.client.Do(req)
	if err != nil {
		return []string{}, err
	}

	//troubleshoot resp err if present
	if resp.StatusCode != 200 {
		//get error code
		var respErr defaultError
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return []string{}, err
		}
		err = json.Unmarshal(body, &respErr)
		if err != nil {
			return []string{}, err
		}
		if respErr.Error.Code == "InvalidAuthenticationToken" {
			return []string{}, fmt.Errorf("access token invalid")
		}
		return []string{}, fmt.Errorf("got non 200 status code from request, got %d", resp.StatusCode)
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
func (m *m365Client) GetJSON(ctx context.Context, endpoint string) ([]jsonLogs, error) {
	// make request to Audit endpoint
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return []jsonLogs{}, err
	}
	req.Header.Set("Authorization", "Bearer "+m.token)
	resp, err := m.client.Do(req)
	if err != nil {
		return []jsonLogs{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	// troubleshoot error code
	if resp.StatusCode != 200 {
		var respErr jsonError
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return []jsonLogs{}, err
		}
		err = json.Unmarshal(body, &respErr)
		if err != nil {
			return []jsonLogs{}, err
		}
		if respErr.Message == "Authorization has been denied for this request." {
			return []jsonLogs{}, fmt.Errorf("authorization denied")
		}
		return []jsonLogs{}, fmt.Errorf("got non 200 status code from request, got %d", resp.StatusCode)
	}

	// check if contentUri field if present
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []jsonLogs{}, err
	}
	if len(body) == 0 { // if body is empty, no new log data available
		return []jsonLogs{}, nil
	}

	// extract contentUri field
	var contentUri logResp
	err = json.Unmarshal(body, &contentUri)
	if err != nil {
		return []jsonLogs{}, err
	}

	// read in json
	body, err = m.followLink(ctx, &contentUri)
	if err != nil {
		return []jsonLogs{}, err
	}
	var logData []jsonLogs
	err = json.Unmarshal(body, &logData)
	if err != nil {
		return []jsonLogs{}, err
	}

	return logData, nil
}

// followLink will follow the response of a first request that has a link to the actual content
func (m *m365Client) followLink(ctx context.Context, lr *logResp) ([]byte, error) {
	redirectReq, err := http.NewRequestWithContext(ctx, "GET", lr.Content, nil)
	if err != nil {
		return []byte{}, err
	}
	redirectReq.Header.Set("Authorization", "Bearer "+m.token)
	redirectResp, err := m.client.Do(redirectReq)
	if err != nil {
		return []byte{}, err
	}
	defer func() { _ = redirectResp.Body.Close() }()

	// troubleshoot error code
	if redirectResp.StatusCode != 200 {
		var respErr jsonError
		body, err := io.ReadAll(redirectResp.Body)
		if err != nil {
			return []byte{}, err
		}
		err = json.Unmarshal(body, &respErr)
		if err != nil {
			return []byte{}, err
		}
		if respErr.Message == "Authorization has been denied for this request." {
			return []byte{}, fmt.Errorf("authorization denied")
		}
		return []byte{}, fmt.Errorf("got non 200 status code from request, got %d", redirectResp.StatusCode)
	}

	return io.ReadAll(redirectResp.Body)
}

func (m *m365Client) StartSubscription(endpoint string) error {
	req, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+m.token)
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// troubleshoot error handling, mainly sub already enabled
	// no error if sub already enabled, not troubleshooting stale token
	// only called while code is synchronous right after a GetToken call
	// if token is stale, don't think regenerating it will fix anything
	if resp.StatusCode != 200 {
		if resp.StatusCode == 400 { // subscription already started possibly
			var respErr defaultError
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			err = json.Unmarshal(body, &respErr)
			if err != nil {
				return err
			}
			if respErr.Error.Message == "The subscription is already enabled. No property change." {
				return nil
			}
		}
		//unsure what the error is
		return fmt.Errorf("got non 200 status code from request, got %d", resp.StatusCode)
	}
	//if StatusCode == 200, then subscription was successfully started
	return nil
}
