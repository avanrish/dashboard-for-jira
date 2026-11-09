// Package jira is a client for the parts of the Jira Cloud REST API the dashboard uses.
package jira

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

// maxRetries limits how often one request is repeated after Jira rate-limits it.
const maxRetries = 3

type Client struct {
	baseURL    *url.URL
	email      string
	apiToken   string
	httpClient *http.Client
}

// NewClient returns a client that authenticates with an API token over basic auth.
// baseURL is the site's address, such as https://example.atlassian.net.
func NewClient(baseURL *url.URL, email, apiToken string) *Client {
	return &Client{
		baseURL:    baseURL,
		email:      email,
		apiToken:   apiToken,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// IssueURL returns the address of an issue's page in Jira.
func (c *Client) IssueURL(key string) string {
	return c.baseURL.JoinPath("browse", key).String()
}

// APIError is a Jira response with a status code outside 2xx.
type APIError struct {
	StatusCode int
	Messages   []string
	RetryAfter time.Duration // set on 429 responses
}

func (e *APIError) Error() string {
	message := fmt.Sprintf("Jira responded %d %s", e.StatusCode, http.StatusText(e.StatusCode))
	if len(e.Messages) > 0 {
		message += fmt.Sprintf(": %s", strings.Join(e.Messages, "; "))
	}
	if e.StatusCode == http.StatusUnauthorized {
		// Jira's messages usually end with a full stop already.
		message = fmt.Sprintf(
			"%s. The API token may have expired: run dashboard-for-jira configure to replace it, or update JIRA_API_TOKEN if you set it.",
			strings.TrimSuffix(message, "."),
		)
	}
	return message
}

// get decodes the JSON response to a GET request into out, waiting and retrying when Jira rate-limits it.
func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	endpoint := c.baseURL.JoinPath(path)
	endpoint.RawQuery = query.Encode()
	for attempt := 0; ; attempt++ {
		err := c.getOnce(ctx, endpoint.String(), out)
		var apiErr *APIError
		if attempt == maxRetries || !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusTooManyRequests {
			return err
		}
		timer := time.NewTimer(apiErr.RetryAfter)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (c *Client) getOnce(ctx context.Context, endpoint string, out any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	request.SetBasicAuth(c.email, c.apiToken)
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return newAPIError(response)
	}
	return json.NewDecoder(response.Body).Decode(out)
}

func newAPIError(response *http.Response) *APIError {
	apiErr := &APIError{StatusCode: response.StatusCode}
	var body struct {
		ErrorMessages []string          `json:"errorMessages"`
		Errors        map[string]string `json:"errors"`
	}
	if json.NewDecoder(io.LimitReader(response.Body, 1<<16)).Decode(&body) == nil {
		apiErr.Messages = body.ErrorMessages
		for _, field := range slices.Sorted(maps.Keys(body.Errors)) {
			apiErr.Messages = append(apiErr.Messages, fmt.Sprintf("%s: %s", field, body.Errors[field]))
		}
	}
	if response.StatusCode == http.StatusTooManyRequests {
		apiErr.RetryAfter = time.Second
		if seconds, err := strconv.Atoi(response.Header.Get("Retry-After")); err == nil && seconds >= 0 {
			apiErr.RetryAfter = time.Duration(seconds) * time.Second
		}
	}
	return apiErr
}
