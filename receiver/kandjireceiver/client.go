package kandjireceiver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.uber.org/zap"
)

type kandjiClient struct {
	client *http.Client

	subdomain string
	region    string
	baseHost  string
	token     string

	baseURL string
	logger  *zap.Logger
}

func newKandjiClient(
	client *http.Client,
	subdomain string,
	region string,
	token string,
	baseHost string,
	logger *zap.Logger,
) *kandjiClient {
	if client == nil {
		client = http.DefaultClient
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	// Normalize
	sub := strings.ToLower(subdomain)
	reg := strings.ToUpper(region)

	// Default Kandji API host selection
	if baseHost == "" {
		switch reg {
		case "EU":
			baseHost = "api.eu.kandji.io"
		case "US", "":
			baseHost = "api.kandji.io"
		default:
			baseHost = "api.kandji.io" // fallback
		}
	}

	// Final API URL:
	//   https://{subdomain}.{baseHost}/api/v1
	baseURL := fmt.Sprintf("https://%s.%s/api/v1", sub, baseHost)

	return &kandjiClient{
		client:    client,
		subdomain: sub,
		region:    reg,
		baseHost:  baseHost,
		token:     token,
		baseURL:   baseURL,
		logger:    logger,
	}
}

func (c *kandjiClient) Shutdown() error {
	c.client.CloseIdleConnections()
	return nil
}

func (c *kandjiClient) CallAPI(
	ctx context.Context,
	ep KandjiEndpoint,
	params map[string]any,
	out any,
) (int, error) {

	spec, ok := EndpointRegistry[ep]
	if !ok {
		return 0, fmt.Errorf("unknown endpoint: %s", ep)
	}

	if err := ValidateParams(ep, params); err != nil {
		return 0, err
	}

	fullURL, err := BuildURL(c.baseURL, spec, params)
	if err != nil {
		return 0, err
	}

	c.logger.Info("kandji api request",
		zap.String("endpoint", string(ep)),
		zap.String("method", spec.Method),
		zap.String("url", fullURL),
		zap.Any("params", params),
	)

	req, err := http.NewRequestWithContext(ctx, spec.Method, fullURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token) // Use 'c.token'
	req.Header.Set("Accept", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, err
	}

	if res.StatusCode >= 400 {
		err = fmt.Errorf("kandji API error: HTTP %d: %s", res.StatusCode, string(bodyBytes))
		c.logResponse(ep, res.StatusCode, len(bodyBytes), bodyBytes, err)
		return res.StatusCode, err
	}

	c.logResponse(ep, res.StatusCode, len(bodyBytes), bodyBytes, nil)

	if out != nil {
		if err := json.Unmarshal(bodyBytes, out); err != nil {
			return res.StatusCode, err
		}
	}

	return res.StatusCode, nil
}

func (c *kandjiClient) logResponse(
	ep KandjiEndpoint,
	status int,
	size int,
	body []byte,
	callErr error,
) {
	fields := []zap.Field{
		zap.String("endpoint", string(ep)),
		zap.Int("status", status),
		zap.Int("bytes", size),
	}
	if callErr != nil {
		fields = append(fields, zap.Error(callErr))
	}
	fields = append(fields, zap.String("body_preview", truncateBody(body, 512)))
	c.logger.Info("kandji api response", fields...)
}

func truncateBody(b []byte, limit int) string {
	if len(b) <= limit {
		return string(b)
	}
	return string(b[:limit]) + "...(truncated)"
}

// BuildURL remains exactly the same
func BuildURL(baseURL string, spec EndpointSpec, params map[string]any) (string, error) {
	full := baseURL + spec.Path

	q := url.Values{}
	for _, p := range spec.Params {
		if v, ok := params[p.Name]; ok {
			q.Set(p.Name, fmt.Sprintf("%v", v))
		}
	}

	if len(q) > 0 {
		return full + "?" + q.Encode(), nil
	}
	return full, nil
}
