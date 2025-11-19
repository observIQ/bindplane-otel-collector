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

package restapireceiver

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

// restAPIClient is an interface for making REST API requests.
// This interface allows for easier testing by enabling mock implementations.
type restAPIClient interface {
	// GetJSON fetches JSON data from the specified URL with the given query parameters.
	// Returns an array of map[string]any representing the JSON objects.
	GetJSON(ctx context.Context, requestURL string, params url.Values) ([]map[string]any, error)
	// GetFullResponse fetches the full JSON response from the specified URL.
	// Returns the full response as map[string]any for pagination parsing.
	GetFullResponse(ctx context.Context, requestURL string, params url.Values) (map[string]any, error)
}

// defaultRESTAPIClient is the default implementation of restAPIClient.
type defaultRESTAPIClient struct {
	client        *http.Client
	cfg           *Config
	logger        *zap.Logger
	responseField string
}

// newRESTAPIClient creates a new REST API client.
func newRESTAPIClient(
	ctx context.Context,
	settings component.TelemetrySettings,
	cfg *Config,
	host component.Host,
) (restAPIClient, error) {
	httpClient, err := cfg.ClientConfig.ToClient(ctx, host, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	return &defaultRESTAPIClient{
		client:        httpClient,
		cfg:           cfg,
		logger:        settings.Logger,
		responseField: cfg.ResponseField,
	}, nil
}

// GetJSON fetches JSON data from the specified URL with the given query parameters.
func (c *defaultRESTAPIClient) GetJSON(ctx context.Context, requestURL string, params url.Values) ([]map[string]any, error) {
	// Build the request URL with query parameters
	u, err := url.Parse(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Add query parameters
	if len(params) > 0 {
		existingParams := u.Query()
		for key, values := range params {
			for _, value := range values {
				existingParams.Add(key, value)
			}
		}
		u.RawQuery = existingParams.Encode()
	}

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply authentication
	if err := c.applyAuth(req); err != nil {
		return nil, fmt.Errorf("failed to apply authentication: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check for HTTP errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse JSON
	var jsonData any
	if err := jsoniter.Unmarshal(body, &jsonData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Extract the array from the response
	var dataArray []any
	if c.responseField != "" {
		// Response has a field containing the array
		responseMap, ok := jsonData.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("response is not a JSON object when response_field is set")
		}
		fieldValue, ok := responseMap[c.responseField]
		if !ok {
			return nil, fmt.Errorf("response field '%s' not found in response", c.responseField)
		}
		dataArray, ok = fieldValue.([]any)
		if !ok {
			return nil, fmt.Errorf("response field '%s' is not an array", c.responseField)
		}
	} else {
		// Response is directly an array
		var ok bool
		dataArray, ok = jsonData.([]any)
		if !ok {
			return nil, fmt.Errorf("response is not a JSON array")
		}
	}

	// Convert []any to []map[string]any
	result := make([]map[string]any, 0, len(dataArray))
	for _, item := range dataArray {
		itemMap, ok := item.(map[string]any)
		if !ok {
			c.logger.Warn("skipping non-object item in array", zap.Any("item", item))
			continue
		}
		result = append(result, itemMap)
	}

	return result, nil
}

// GetFullResponse fetches the full JSON response from the specified URL.
func (c *defaultRESTAPIClient) GetFullResponse(ctx context.Context, requestURL string, params url.Values) (map[string]any, error) {
	// Build the request URL with query parameters
	u, err := url.Parse(requestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Add query parameters
	if len(params) > 0 {
		existingParams := u.Query()
		for key, values := range params {
			for _, value := range values {
				existingParams.Add(key, value)
			}
		}
		u.RawQuery = existingParams.Encode()
	}

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply authentication
	if err := c.applyAuth(req); err != nil {
		return nil, fmt.Errorf("failed to apply authentication: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check for HTTP errors
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse JSON
	var jsonData any
	if err := jsoniter.Unmarshal(body, &jsonData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Return as map
	responseMap, ok := jsonData.(map[string]any)
	if !ok {
		// If response is an array, wrap it in a map
		if arr, ok := jsonData.([]any); ok {
			return map[string]any{"data": arr}, nil
		}
		return nil, fmt.Errorf("response is not a JSON object or array")
	}

	return responseMap, nil
}

// generateEdgeGridAuth generates the Akamai EdgeGrid authentication header.
func (c *defaultRESTAPIClient) generateEdgeGridAuth(req *http.Request) (string, error) {
	// Generate timestamp in ISO 8601 format
	timestamp := time.Now().UTC().Format("20060102T15:04:05+0000")

	// Generate nonce (random UUID-like string)
	nonce, err := generateNonce()
	if err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Create the data to sign
	// Format: timestamp + "\t" + nonce + "\t" + method + "\t" + path + query + "\t" + headers + "\t" + body
	path := req.URL.Path
	if req.URL.RawQuery != "" {
		path += "?" + req.URL.RawQuery
	}

	// For GET requests, body is empty
	body := ""

	// Construct the signing data
	signingData := strings.Join([]string{
		timestamp,
		nonce,
		req.Method,
		path,
		"", // headers (usually empty)
		body,
	}, "\t")

	// Create signing key from client secret and timestamp
	signingKey := makeSigningKey(timestamp, c.cfg.AkamaiEdgeGridConfig.ClientSecret)

	// Create the signature
	signature := makeSignature(signingData, signingKey)

	// Construct the authorization header
	authHeader := fmt.Sprintf(
		"EG1-HMAC-SHA256 client_token=%s;access_token=%s;timestamp=%s;nonce=%s;signature=%s",
		c.cfg.AkamaiEdgeGridConfig.ClientToken,
		c.cfg.AkamaiEdgeGridConfig.AccessToken,
		timestamp,
		nonce,
		signature,
	)

	return authHeader, nil
}

// generateNonce generates a random nonce for the EdgeGrid request.
func generateNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

// makeSigningKey creates the signing key from the timestamp and client secret.
func makeSigningKey(timestamp, clientSecret string) string {
	mac := hmac.New(sha256.New, []byte(clientSecret))
	mac.Write([]byte(timestamp))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// makeSignature creates the HMAC-SHA256 signature.
func makeSignature(data, key string) string {
	keyBytes, _ := base64.StdEncoding.DecodeString(key)
	mac := hmac.New(sha256.New, keyBytes)
	mac.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// applyAuth applies authentication headers to the request based on the configured auth mode.
func (c *defaultRESTAPIClient) applyAuth(req *http.Request) error {
	switch c.cfg.AuthMode {

	case authModeAPIKey:
		// API key authentication
		if c.cfg.APIKeyConfig.HeaderName == "" || c.cfg.APIKeyConfig.Value == "" {
			return fmt.Errorf("API key header name and value are required")
		}
		req.Header.Set(c.cfg.APIKeyConfig.HeaderName, c.cfg.APIKeyConfig.Value)
		return nil

	case authModeBearer:
		// Bearer token authentication
		if c.cfg.BearerConfig.Token == "" {
			return fmt.Errorf("bearer token is required")
		}
		req.Header.Set("Authorization", "Bearer "+c.cfg.BearerConfig.Token)
		return nil

	case authModeBasic:
		// Basic authentication
		if c.cfg.BasicConfig.Username == "" || c.cfg.BasicConfig.Password == "" {
			return fmt.Errorf("basic auth username and password are required")
		}
		req.SetBasicAuth(c.cfg.BasicConfig.Username, c.cfg.BasicConfig.Password)
		return nil

	case authModeAkamaiEdgeGrid:
		// Akamai EdgeGrid authentication
		if c.cfg.AkamaiEdgeGridConfig.AccessToken == "" || c.cfg.AkamaiEdgeGridConfig.ClientToken == "" || c.cfg.AkamaiEdgeGridConfig.ClientSecret == "" {
			return fmt.Errorf("akamai edgegrid access token, client token, and client secret are required")
		}
		authHeader, err := c.generateEdgeGridAuth(req)
		if err != nil {
			return fmt.Errorf("failed to generate EdgeGrid auth: %w", err)
		}
		req.Header.Set("Authorization", authHeader)
		return nil

	default:
		return fmt.Errorf("unsupported auth mode: %s", c.cfg.AuthMode)
	}
}
