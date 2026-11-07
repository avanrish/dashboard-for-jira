package jira

import (
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
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
		"priority": {"name": "High"}, "assignee": {"displayName": "Ada Lovelace"}, "customfield_10001": {"id": "36885b3c", "name": "Platform"}}}
]}`

// The last page holds a subtask of DASH-2, a story whose epic is not on the board, and an issue in a status without a column.
const lastIssuePageJSON = `{"issues": [
	{"key": "DASH-4", "fields": {"summary": "Configure the identity provider", "status": {"id": "3"}, "issuetype": {"name": "Subtask", "hierarchyLevel": -1},
		"parent": {"id": "102", "key": "DASH-2", "fields": {"summary": "Add SSO", "issuetype": {"name": "Story", "hierarchyLevel": 0}}}}},
	{"key": "DASH-5", "fields": {"summary": "Cache dependencies", "status": {"id": "1"}, "issuetype": {"name": "Task", "hierarchyLevel": 0},
		"parent": {"id": "109", "key": "DASH-9", "fields": {"summary": "Speed up CI", "issuetype": {"name": "Epic", "hierarchyLevel": 1}}}}},
	{"key": "DASH-6", "fields": {"summary": "Unmapped status", "status": {"id": "99"}, "issuetype": {"name": "Task", "hierarchyLevel": 0}}}
]}`

func TestBoardGroupsIssuesIntoEpicLanes(t *testing.T) {
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
		if got, want := r.URL.Query().Get("fields"), "summary,status,issuetype,priority,assignee,parent,customfield_10001"; got != want {
			t.Errorf("fields = %q, want %q", got, want)
		}
		// The backlog's status 10 is left out.
		if got, want := r.URL.Query().Get("jql"), "(fixVersion is EMPTY) AND status in (1, 3, 5) AND ("+recentlyDone+")"; got != want {
			t.Errorf("jql = %q, want %q", got, want)
		}
		if r.URL.Query().Get("nextPageToken") == "page-2" {
			_, _ = io.WriteString(w, lastIssuePageJSON)
		} else {
			_, _ = io.WriteString(w, firstIssuePageJSON)
		}
	})
	mux.HandleFunc("GET /rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Query().Get("jql"), "id in (101, 109) ORDER BY Rank ASC"; got != want {
			t.Errorf("jql = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("fields"), "customfield_10002"; got != want {
			t.Errorf("fields = %q, want %q", got, want)
		}
		_, _ = io.WriteString(w, `{"issues": [
			{"id": "109", "fields": {"customfield_10002": "purple"}},
			{"id": "101", "fields": {"customfield_10002": "ghx-label-11"}}
		]}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	source := BoardSource{Client: newTestClient(t, server.URL), BoardID: 7}
	got, err := source.Board(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if got.Name != "Team board" || !slices.Equal(got.Columns, []string{"To Do", "In Progress", "Done"}) {
		t.Errorf("board = %q with columns %q", got.Name, got.Columns)
	}
	var lanes []string
	for _, lane := range got.Lanes {
		lanes = append(lanes, describeLane(lane))
	}
	// Lanes follow epic rank, and the empty epic DASH-10 gets none.
	wantLanes := []string{
		"DASH-9 Speed up CI: [DASH-5] [] []",
		"DASH-1 Auth revamp: [DASH-2] [DASH-4] []",
		" : [] [] [DASH-3]",
	}
	if !slices.Equal(lanes, wantLanes) {
		t.Errorf("lanes =\n%s\nwant\n%s", strings.Join(lanes, "\n"), strings.Join(wantLanes, "\n"))
	}
	if colors := []string{got.Lanes[0].Epic.Color, got.Lanes[1].Epic.Color}; !slices.Equal(colors, []string{"purple", "teal"}) {
		t.Errorf("epic colours = %q, want purple and teal", colors)
	}
	wantCard := board.Issue{
		Key: "DASH-3", Summary: "Fix login", Type: "Bug", Status: "Done", StatusCategory: board.Done,
		Priority: "High", Assignee: "Ada Lovelace", Team: "Platform", URL: server.URL + "/browse/DASH-3",
	}
	if card := got.Lanes[2].Cells[2][0]; card != wantCard {
		t.Errorf("card = %+v, want %+v", card, wantCard)
	}
}

func describeLane(lane board.Lane) string {
	var cells []string
	for _, issues := range lane.Cells {
		var keys []string
		for _, issue := range issues {
			keys = append(keys, issue.Key)
		}
		cells = append(cells, "["+strings.Join(keys, " ")+"]")
	}
	return lane.Epic.Key + " " + lane.Epic.Summary + ": " + strings.Join(cells, " ")
}
