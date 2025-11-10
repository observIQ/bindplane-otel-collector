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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

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
	if err := json.Unmarshal(body, &jsonData); err != nil {
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
	if err := json.Unmarshal(body, &jsonData); err != nil {
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

// applyAuth applies authentication headers to the request based on the configured auth mode.
func (c *defaultRESTAPIClient) applyAuth(req *http.Request) error {
	switch c.cfg.Auth.Mode {
	case AuthModeNone:
		// No authentication
		return nil

	case AuthModeAPIKey:
		// API key authentication
		if c.cfg.Auth.APIKey.HeaderName == "" || c.cfg.Auth.APIKey.Value == "" {
			return fmt.Errorf("API key header name and value are required")
		}
		req.Header.Set(c.cfg.Auth.APIKey.HeaderName, c.cfg.Auth.APIKey.Value)
		return nil

	case AuthModeBearer:
		// Bearer token authentication
		if c.cfg.Auth.BearerToken == "" {
			return fmt.Errorf("bearer token is required")
		}
		req.Header.Set("Authorization", "Bearer "+c.cfg.Auth.BearerToken)
		return nil

	case AuthModeBasic:
		// Basic authentication
		if c.cfg.Auth.BasicAuth.Username == "" || c.cfg.Auth.BasicAuth.Password == "" {
			return fmt.Errorf("basic auth username and password are required")
		}
		req.SetBasicAuth(c.cfg.Auth.BasicAuth.Username, c.cfg.Auth.BasicAuth.Password)
		return nil

	default:
		return fmt.Errorf("unsupported auth mode: %s", c.cfg.Auth.Mode)
	}
}
