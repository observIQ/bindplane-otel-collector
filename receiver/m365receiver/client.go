package m365receiver

import (
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
}

type response struct {
	Token string `json:"access_token"`
}
type tokenError struct {
	Err string `json:"error"`
}
type csvError struct {
	ErrorCSV struct {
		Code string `json:"code"`
	} `json:"error"`
}

func newM365Client(c *http.Client, cfg *Config) *m365Client {
	return &m365Client{
		client:       c,
		authEndpoint: fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", cfg.TenantID),
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
	}
}

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
		var respErr csvError
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return []string{}, err
		}
		err = json.Unmarshal(body, &respErr)
		if err != nil {
			return []string{}, err
		}
		if respErr.ErrorCSV.Code == "InvalidAuthenticationToken" {
			return []string{}, fmt.Errorf("Access token invalid.")
		}
		return []string{}, fmt.Errorf("Got non 200 status code from request, got %d.", resp.StatusCode)
	}

	defer resp.Body.Close()
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
		} else {
			return []string{}, err
		}
	}

	return data, nil
}

// Get authorization token
func (m *m365Client) GetToken() error {
	formData := url.Values{
		"grant_type":    {"client_credentials"},
		"scope":         {"https://graph.microsoft.com/.default"},
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

	defer resp.Body.Close()

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
			return fmt.Errorf("The provided client_id is incorrect or does not exist within the given tenant directory.")
		case "invalid_client":
			return fmt.Errorf("The provided client_secret is incorrect or does not belong to the given client_id.")
		case "invalid_request":
			return fmt.Errorf("The provided tenant_id is incorrect or does not exist.")
		}
		return fmt.Errorf("Got non 200 status code from request, got %d.", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var token response
	err = json.Unmarshal(body, &token)
	if err != nil {
		return err
	}

	m.token = token.Token

	return nil
}
