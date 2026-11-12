package jira

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/avanrish/dashboard-for-jira/internal/board"
)

const configurationJSON = `{
	"name": "Team board",
	"type": "kanban",
	"columnConfig": {"columns": [
		{"name": "Backlog", "statuses": [{"id": "10"}]},
		{"name": "To Do", "statuses": [{"id": "1"}]},
		{"name": "In Progress", "statuses": [{"id": "3"}]},
		{"name": "Done", "statuses": [{"id": "5"}]}
	]},
	"subQuery": {"query": "fixVersion is EMPTY"}
}`

// The first page holds an epic with an issue, an epic without one, a story in the first epic, and a bug without an epic.
const firstIssuePageJSON = `{"nextPageToken": "page-2", "issues": [
	{"key": "DASH-1", "fields": {"summary": "Auth revamp", "status": {"id": "3"}, "issuetype": {"name": "Epic", "hierarchyLevel": 1}}},
	{"key": "DASH-10", "fields": {"summary": "Empty epic", "status": {"id": "1"}, "issuetype": {"name": "Epic", "hierarchyLevel": 1}}},
	{"key": "DASH-2", "fields": {"summary": "Add SSO", "status": {"id": "1"}, "issuetype": {"name": "Story", "hierarchyLevel": 0},
		"parent": {"id": "101", "key": "DASH-1", "fields": {"summary": "Auth revamp", "issuetype": {"name": "Epic", "hierarchyLevel": 1}}}}},
	{"key": "DASH-3", "fields": {"summary": "Fix login", "status": {"id": "5", "name": "Done", "statusCategory": {"key": "done"}}, "issuetype": {"name": "Bug", "hierarchyLevel": 0},
		"priority": {"name": "High"}, "assignee": {"displayName": "Ada Lovelace"}, "customfield_10001": {"id": "36885b3c", "name": "Platform"},
		"flagged": true}}
]}`

// The last page holds a subtask of DASH-2, a story whose epic is not on the board, an issue in a status without a column,
// and a subtask of DASH-8, a story in that epic which waits in the backlog.
const lastIssuePageJSON = `{"issues": [
	{"key": "DASH-4", "fields": {"summary": "Configure the identity provider", "status": {"id": "3"}, "issuetype": {"name": "Subtask", "hierarchyLevel": -1},
		"parent": {"id": "102", "key": "DASH-2", "fields": {"summary": "Add SSO", "issuetype": {"name": "Story", "hierarchyLevel": 0}}}}},
	{"key": "DASH-5", "fields": {"summary": "Cache dependencies", "status": {"id": "1"}, "issuetype": {"name": "Task", "hierarchyLevel": 0},
		"parent": {"id": "109", "key": "DASH-9", "fields": {"summary": "Speed up CI", "issuetype": {"name": "Epic", "hierarchyLevel": 1}}}}},
	{"key": "DASH-6", "fields": {"summary": "Unmapped status", "status": {"id": "99"}, "issuetype": {"name": "Task", "hierarchyLevel": 0}}},
	{"key": "DASH-7", "fields": {"summary": "Cache Go modules", "status": {"id": "3"}, "issuetype": {"name": "Subtask", "hierarchyLevel": -1},
		"parent": {"id": "108", "key": "DASH-8", "fields": {"summary": "Speed up builds", "issuetype": {"name": "Story", "hierarchyLevel": 0}}}}}
]}`

// newBoardServer serves board 7 with the issues above, and returns a function that counts the requests to a path.
func newBoardServer(t *testing.T) (*httptest.Server, func(path string) int) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /rest/agile/1.0/board/7/configuration", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, configurationJSON)
	})
	mux.HandleFunc("GET /rest/api/3/field", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[
			{"id": "summary", "schema": {"type": "string", "system": "summary"}},
			{"id": "customfield_10001", "schema": {"type": "any", "custom": "com.atlassian.teams:rm-teams-custom-field-team"}},
			{"id": "customfield_10002", "schema": {"type": "string", "custom": "com.pyxis.greenhopper.jira:jsw-issue-color"}}
		]`)
	})
	mux.HandleFunc("GET /rest/software/1.0/board/7/issue", func(w http.ResponseWriter, r *http.Request) {
		if email, token, _ := r.BasicAuth(); email != "ada@example.com" || token != "test-token" {
			t.Errorf("basic auth = %q, %q", email, token)
		}
		if got, want := r.URL.Query().Get("fields"), "summary,status,issuetype,priority,assignee,parent,flagged,customfield_10001"; got != want {
			t.Errorf("fields = %q, want %q", got, want)
		}
		// The backlog's status 10 is left out.
		if got, want := r.URL.Query().Get("jql"), fmt.Sprintf("(fixVersion is EMPTY) AND status in (1, 3, 5) AND (%s)", recentlyDone); got != want {
			t.Errorf("jql = %q, want %q", got, want)
		}
		if r.URL.Query().Get("nextPageToken") == "page-2" {
			_, _ = io.WriteString(w, lastIssuePageJSON)
		} else {
			_, _ = io.WriteString(w, firstIssuePageJSON)
		}
	})
	mux.HandleFunc("GET /rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		switch jql, fields := r.URL.Query().Get("jql"), r.URL.Query().Get("fields"); jql {
		case "id in (108)": // DASH-7's parent, for its epic
			if fields != "parent" {
				t.Errorf("fields = %q, want parent", fields)
			}
			_, _ = io.WriteString(w, `{"issues": [
				{"id": "108", "key": "DASH-8", "fields": {
					"parent": {"id": "109", "key": "DASH-9", "fields": {"summary": "Speed up CI", "issuetype": {"name": "Epic", "hierarchyLevel": 1}}}}}
			]}`)
		case "id in (101, 109) ORDER BY Rank ASC":
			if fields != "customfield_10002" {
				t.Errorf("fields = %q, want customfield_10002", fields)
			}
			_, _ = io.WriteString(w, `{"issues": [
				{"id": "109", "fields": {"customfield_10002": "purple"}},
				{"id": "101", "fields": {"customfield_10002": "ghx-label-11"}}
			]}`)
		default:
			t.Errorf("unexpected jql %q", jql)
			http.Error(w, "unexpected jql", http.StatusBadRequest)
		}
	})
	var mu sync.Mutex
	counts := make(map[string]int)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		counts[r.URL.Path]++
		mu.Unlock()
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	return server, func(path string) int {
		mu.Lock()
		defer mu.Unlock()
		return counts[path]
	}
}

func TestBoardGroupsIssuesByEpic(t *testing.T) {
	server, _ := newBoardServer(t)
	source := BoardSource{Client: newTestClient(t, server.URL), BoardID: 7}
	got, err := source.Board(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if got.Name != "Team board" || !slices.Equal(got.Columns, []string{"To Do", "In Progress", "Done"}) {
		t.Errorf("board = %q with columns %q", got.Name, got.Columns)
	}
	var epics []string
	for _, epic := range got.Epics {
		epics = append(epics, describeEpic(epic))
	}
	// Epics follow their rank, and the empty epic DASH-10 is left out.
	wantEpics := []string{
		"DASH-9 Speed up CI: [DASH-5] [DASH-7] []",
		"DASH-1 Auth revamp: [DASH-2] [DASH-4] []",
		" : [] [] [DASH-3]",
	}
	if !slices.Equal(epics, wantEpics) {
		t.Errorf("epics =\n%s\nwant\n%s", strings.Join(epics, "\n"), strings.Join(wantEpics, "\n"))
	}
	if colors := []string{got.Epics[0].Color, got.Epics[1].Color}; !slices.Equal(colors, []string{"purple", "teal"}) {
		t.Errorf("epic colours = %q, want purple and teal", colors)
	}
	cardURL, err := url.JoinPath(server.URL, "browse", "DASH-3")
	if err != nil {
		t.Fatal(err)
	}
	wantCard := board.Issue{
		Key: "DASH-3", Summary: "Fix login", Type: "Bug", Status: "Done", StatusCategory: board.Done,
		Priority: "High", Assignee: "Ada Lovelace", Team: "Platform", Flagged: true, URL: cardURL,
	}
	if card := got.Epics[2].Cells[2][0]; card != wantCard {
		t.Errorf("card = %+v, want %+v", card, wantCard)
	}
}

// A reload fetches the issues again, but reuses the cached settings and searches.
func TestBoardReloadFetchesOnlyIssues(t *testing.T) {
	server, requests := newBoardServer(t)
	source := BoardSource{Client: newTestClient(t, server.URL), BoardID: 7}
	for range 2 {
		if _, err := source.Board(t.Context()); err != nil {
			t.Fatal(err)
		}
	}

	for _, want := range []struct {
		path     string
		requests int
	}{
		{"/rest/agile/1.0/board/7/configuration", 1},
		{"/rest/api/3/field", 1},
		{"/rest/software/1.0/board/7/issue", 4}, // two pages, twice
		{"/rest/api/3/search/jql", 2},           // the subtasks' parents and the epics' order, once each
	} {
		if got := requests(want.path); got != want.requests {
			t.Errorf("%s: %d requests, want %d", want.path, got, want.requests)
		}
	}
}

// With an expired token, both settings requests fail the same way; the error should explain that once.
func TestBoardErrorExplainsAFailureOnce(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := BoardSource{Client: newTestClient(t, server.URL), BoardID: 7}.Board(t.Context())
	if err == nil {
		t.Fatal("got no error, want a 401")
	}
	if n := strings.Count(err.Error(), "API token may have expired"); n != 1 {
		t.Errorf("error = %q, explains the expired token %d times, want once", err, n)
	}
}

func describeEpic(epic board.Epic) string {
	var cells []string
	for _, issues := range epic.Cells {
		var keys []string
		for _, issue := range issues {
			keys = append(keys, issue.Key)
		}
		cells = append(cells, fmt.Sprintf("[%s]", strings.Join(keys, " ")))
	}
	return fmt.Sprintf("%s %s: %s", epic.Key, epic.Summary, strings.Join(cells, " "))
}
