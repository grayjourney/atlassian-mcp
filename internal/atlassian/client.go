// Package atlassian is a thin REST client for Atlassian Cloud (Jira + Confluence)
// using basic auth (email + API token). It deliberately models only the fields
// the MVP tools surface; everything else is passed through as raw JSON so the
// tool layer stays in control of what the LLM sees.
package atlassian

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// APIError represents a non-2xx response from an Atlassian API.
type APIError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *APIError) Error() string {
	body := e.Body
	if len(body) > 500 {
		body = body[:500] + "…"
	}
	return fmt.Sprintf("atlassian API error %d (%s): %s", e.StatusCode, e.Status, body)
}

// restClient performs authenticated JSON requests against one Atlassian base URL.
type restClient struct {
	baseURL    string
	authHeader string
	httpClient *http.Client
}

func newRESTClient(baseURL, username, token string, hc *http.Client) *restClient {
	if hc == nil {
		hc = &http.Client{Timeout: 75 * time.Second}
	}
	cred := base64.StdEncoding.EncodeToString([]byte(username + ":" + token))
	return &restClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		authHeader: "Basic " + cred,
		httpClient: hc,
	}
}

// do executes method against path (joined onto baseURL) with optional query and
// JSON body, decoding a JSON response into out when out is non-nil. A non-2xx
// status is returned as *APIError.
func (c *restClient) do(ctx context.Context, method, path string, query url.Values, body, out any) error {
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Status: resp.Status, Body: string(raw)}
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

// getBytes GETs an absolute URL with auth and returns the raw body and its
// Content-Type. Used for binary downloads (attachments) where the API hands
// back a full content URL rather than a path.
func (c *restClient) getBytes(ctx context.Context, fullURL string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("GET %s: %w", fullURL, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", &APIError{StatusCode: resp.StatusCode, Status: resp.Status, Body: string(raw)}
	}
	return raw, resp.Header.Get("Content-Type"), nil
}
