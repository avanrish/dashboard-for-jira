package jira

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func newTestClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	baseURL, err := url.Parse(serverURL)
	if err != nil {
		t.Fatal(err)
	}
	return NewClient(baseURL, "ada@example.com", "test-token")
}

func TestRateLimitedRequestIsRetried(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = io.WriteString(w, configurationJSON)
	}))
	defer server.Close()

	configuration, err := newTestClient(t, server.URL).BoardConfiguration(t.Context(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 2 || configuration.Name != "Team board" {
		t.Errorf("got %q after %d requests, want Team board after 2", configuration.Name, requests.Load())
	}
}

func TestErrorShowsJiraMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"errorMessages": ["Client must be authenticated to access this resource."]}`)
	}))
	defer server.Close()

	_, err := newTestClient(t, server.URL).BoardConfiguration(t.Context(), 7)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("error = %v, want a 401 APIError", err)
	}
	for _, want := range []string{"Client must be authenticated", "API token may have expired"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want it to contain %q", err, want)
		}
	}
}
