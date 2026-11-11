package jira

import (
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// stopClock makes the client's cache see a clock that moves only when the returned function advances it.
func stopClock(client *Client) (advance func(time.Duration)) {
	var mu sync.Mutex
	now := time.Now()
	client.cache.now = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	}
	return func(d time.Duration) {
		mu.Lock()
		defer mu.Unlock()
		now = now.Add(d)
	}
}

func TestRequestReusesResponseForItsTTL(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = io.WriteString(w, configurationJSON)
	}))
	defer server.Close()
	client := newTestClient(t, server.URL)
	advance := stopClock(client)

	steps := []struct {
		description  string
		wait         time.Duration // before the request
		options      RequestOptions
		wantRequests int32 // in total, after the request
	}{
		{"first request", 0, RequestOptions{CacheTTL: 5 * time.Minute}, 1},
		{"within its TTL", 5*time.Minute - time.Second, RequestOptions{CacheTTL: 5 * time.Minute}, 1},
		{"after its TTL", time.Second, RequestOptions{CacheTTL: 5 * time.Minute}, 2},
		{"within a longer TTL", 4 * time.Minute, RequestOptions{CacheTTL: time.Hour}, 2},
		{"older than a shorter TTL", 0, RequestOptions{CacheTTL: time.Minute}, 3},
		{"without a TTL", 0, RequestOptions{}, 4},
	}
	for _, step := range steps {
		advance(step.wait)
		configuration, err := client.BoardConfiguration(t.Context(), 7, step.options)
		if err != nil {
			t.Fatalf("%s: %v", step.description, err)
		}
		if configuration.Name != "Team board" || requests.Load() != step.wantRequests {
			t.Errorf("%s: got %q after %d requests, want Team board after %d", step.description, configuration.Name, requests.Load(), step.wantRequests)
		}
	}
}

func TestFailedResponseIsNotCached(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = io.WriteString(w, configurationJSON)
	}))
	defer server.Close()
	client := newTestClient(t, server.URL)
	options := RequestOptions{CacheTTL: time.Hour}

	if _, err := client.BoardConfiguration(t.Context(), 7, options); err == nil {
		t.Fatal("first request succeeded, want the 500 error")
	}
	configuration, err := client.BoardConfiguration(t.Context(), 7, options)
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 2 || configuration.Name != "Team board" {
		t.Errorf("got %q after %d requests, want Team board after 2", configuration.Name, requests.Load())
	}
}

func TestIssuePagesExpireTogether(t *testing.T) {
	var advance func(time.Duration)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Query().Get("nextPageToken") == "page-2" {
			// Jira takes a second to send the last page.
			advance(time.Second)
			_, _ = io.WriteString(w, `{"issues": [{"key": "DASH-2"}]}`)
			return
		}
		_, _ = io.WriteString(w, `{"nextPageToken": "page-2", "issues": [{"key": "DASH-1"}]}`)
	}))
	defer server.Close()
	client := newTestClient(t, server.URL)
	advance = stopClock(client)

	steps := []struct {
		description  string
		wait         time.Duration // before the request
		wantRequests int32         // in total, after the request
	}{
		{"first request", 0, 2},
		// Cached page by page, the first page would have expired here while the second had not.
		{"five minutes after the first page arrived", 5*time.Minute - time.Second, 2},
		{"five minutes after the last page arrived", time.Second, 4},
	}
	for _, step := range steps {
		advance(step.wait)
		issues, err := client.SearchIssues(t.Context(), "project = DASH", nil, RequestOptions{CacheTTL: 5 * time.Minute})
		if err != nil {
			t.Fatalf("%s: %v", step.description, err)
		}
		var keys []string
		for _, issue := range issues {
			keys = append(keys, issue.Key)
		}
		if !slices.Equal(keys, []string{"DASH-1", "DASH-2"}) || requests.Load() != step.wantRequests {
			t.Errorf("%s: got %v after %d requests, want [DASH-1 DASH-2] after %d", step.description, keys, requests.Load(), step.wantRequests)
		}
	}
}

func TestCacheDropsExpiredResponses(t *testing.T) {
	now := time.Now()
	cache := responseCache{now: func() time.Time { return now }}
	cache.put("noTTL", nil, 0)
	cache.put("minute", nil, time.Minute)
	cache.put("hour", nil, time.Hour)

	now = now.Add(time.Minute)
	cache.put("new", nil, time.Minute)

	if got, want := slices.Sorted(maps.Keys(cache.entries)), []string{"hour", "new"}; !slices.Equal(got, want) {
		t.Errorf("entries = %v, want %v", got, want)
	}
}

// Myself checks the API token, so it must not answer from the cache after the token is revoked.
func TestMyselfFailsAfterTokenIsRevoked(t *testing.T) {
	var revoked atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if revoked.Load() {
			http.Error(w, `{"errorMessages": ["Unauthorized"]}`, http.StatusUnauthorized)
			return
		}
		_, _ = io.WriteString(w, `{"displayName": "Ada"}`)
	}))
	defer server.Close()
	client := newTestClient(t, server.URL)

	if _, err := client.Myself(t.Context()); err != nil {
		t.Fatal(err)
	}
	revoked.Store(true)
	if name, err := client.Myself(t.Context()); err == nil {
		t.Errorf("Myself = %q after the token was revoked, want an error", name)
	}
}
