// Copyright  observIQ, Inc.
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

package lookupprocessor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	maxRetries     = 3
	retryDelay     = 100 * time.Millisecond
	retryMultiplier = 2
)

// APISource implements LookupSource for REST API endpoints
type APISource struct {
	urlTemplate     string
	method          string
	headers         map[string]string
	timeout         time.Duration
	responseMapping map[string]string
	client          *http.Client
	logger          *zap.Logger
}

// NewAPISource creates a new APISource
func NewAPISource(cfg *APIConfig, logger *zap.Logger) (*APISource, error) {
	client := &http.Client{
		Timeout: cfg.Timeout,
	}

	return &APISource{
		urlTemplate:     cfg.URL,
		method:          cfg.Method,
		headers:         cfg.Headers,
		timeout:         cfg.Timeout,
		responseMapping: cfg.ResponseMapping,
		client:          client,
		logger:          logger,
	}, nil
}

// Lookup makes an API call with the key substituted in the URL
func (a *APISource) Lookup(key string) (map[string]string, error) {
	// Substitute the key in the URL template
	requestURL := a.substituteURL(key)

	// Validate URL
	if _, err := url.Parse(requestURL); err != nil {
		return nil, fmt.Errorf("invalid URL after substitution: %w", err)
	}

	// Make API call with retries
	var lastErr error
	delay := retryDelay

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			a.logger.Debug("retrying API request", zap.Int("attempt", attempt), zap.Duration("delay", delay))
			time.Sleep(delay)
			delay *= retryMultiplier
		}

		data, err := a.makeRequest(requestURL)
		if err == nil {
			return data, nil
		}

		lastErr = err
		a.logger.Debug("API request failed", zap.Error(err), zap.Int("attempt", attempt+1))
	}

	return nil, fmt.Errorf("API request failed after %d attempts: %w", maxRetries, lastErr)
}

// Load is a no-op for API source
func (a *APISource) Load() error {
	return nil
}

// Close cleans up resources
func (a *APISource) Close() error {
	if a.client != nil {
		a.client.CloseIdleConnections()
	}
	return nil
}

// substituteURL replaces placeholders in the URL template with the lookup key
func (a *APISource) substituteURL(key string) string {
	// URL encode the key for safe substitution
	encodedKey := url.QueryEscape(key)
	
	// Replace common placeholder patterns
	result := a.urlTemplate
	result = strings.ReplaceAll(result, "$fieldValue", encodedKey)
	result = strings.ReplaceAll(result, "${fieldValue}", encodedKey)
	result = strings.ReplaceAll(result, "$key", encodedKey)
	result = strings.ReplaceAll(result, "${key}", encodedKey)
	
	return result
}

// makeRequest performs the actual HTTP request
func (a *APISource) makeRequest(requestURL string) (map[string]string, error) {
	req, err := http.NewRequest(a.method, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for key, value := range a.headers {
		req.Header.Set(key, value)
	}

	// Make request
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse JSON response
	var jsonData map[string]interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Apply response mapping if configured
	if len(a.responseMapping) > 0 {
		return a.applyResponseMapping(jsonData)
	}

	// Otherwise, flatten the JSON to map[string]string
	return a.flattenJSON(jsonData)
}

// applyResponseMapping extracts specific fields from JSON based on mapping configuration
func (a *APISource) applyResponseMapping(jsonData map[string]interface{}) (map[string]string, error) {
	result := make(map[string]string)

	for fieldName, jsonPath := range a.responseMapping {
		value, err := a.extractJSONPath(jsonData, jsonPath)
		if err != nil {
			a.logger.Debug("failed to extract JSON path", zap.String("path", jsonPath), zap.Error(err))
			continue
		}
		result[fieldName] = value
	}

	return result, nil
}

// extractJSONPath extracts a value from JSON using a simple path (e.g., "data.hostname")
func (a *APISource) extractJSONPath(data map[string]interface{}, path string) (string, error) {
	parts := strings.Split(path, ".")
	var current interface{} = data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			var ok bool
			current, ok = v[part]
			if !ok {
				return "", fmt.Errorf("path segment '%s' not found", part)
			}
		default:
			return "", fmt.Errorf("cannot navigate through non-object at '%s'", part)
		}
	}

	// Convert final value to string
	return a.valueToString(current), nil
}

// flattenJSON converts a JSON object to map[string]string
func (a *APISource) flattenJSON(data map[string]interface{}) (map[string]string, error) {
	result := make(map[string]string)

	for key, value := range data {
		result[key] = a.valueToString(value)
	}

	return result, nil
}

// valueToString converts various types to string
func (a *APISource) valueToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%v", v)
	case int:
		return fmt.Sprintf("%d", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case nil:
		return ""
	default:
		// For complex types, marshal to JSON
		if bytes, err := json.Marshal(v); err == nil {
			return string(bytes)
		}
		return fmt.Sprintf("%v", v)
	}
}
