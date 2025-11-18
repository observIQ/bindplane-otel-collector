package kandjireceiver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type kandjiClient struct {
	client *http.Client

	subdomain string
	region    string
	baseHost  string
	token     string

	baseURL string
}

func newKandjiClient(
	client *http.Client,
	subdomain string,
	region string,
	token string,
	baseHost string,
) *kandjiClient {
	if client == nil {
		client = http.DefaultClient
	}
	if baseHost == "" {
		baseHost = "kandji.io"
	}

	host := baseHost
	if region != "" && region != "US" {
		host = fmt.Sprintf("%s.%s", region, baseHost)
	}

	baseURL := fmt.Sprintf("https://%s.%s/api/v1", subdomain, host)

	return &kandjiClient{
		client:    client,
		subdomain: subdomain,
		region:    region,
		baseHost:  baseHost,
		token:     token,
		baseURL:   baseURL,
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
) error {

	spec, ok := EndpointRegistry[ep]
	if !ok {
		return fmt.Errorf("unknown endpoint: %s", ep)
	}

	if err := ValidateParams(ep, params); err != nil {
		return err
	}

	fullURL, err := BuildURL(c.baseURL, spec, params)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, spec.Method, fullURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token) // Use 'c.token'
	req.Header.Set("Accept", "application/json")

	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode >= 400 {
		return fmt.Errorf("kandji API error: HTTP %d: %s", res.StatusCode, string(bodyBytes))
	}

	if out != nil {
		if err := json.Unmarshal(bodyBytes, out); err != nil {
			return err
		}
	}

	return nil
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
