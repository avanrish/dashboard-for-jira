package board

import "context"

// MockSource serves a fixed sample board for working on the UI without Jira.
type MockSource struct{}

// Board returns a new copy on each call, so callers may modify it.
func (MockSource) Board(context.Context) (Board, error) {
	return Board{
		Name: "DASH board",
		Columns: []Column{
			{Name: "To Do", Issues: []Issue{
				{Key: "DASH-21", Summary: "Show issue type icons on cards", Type: "Story", Priority: "High"},
				{Key: "DASH-24", Summary: "Board header overlaps the first column on narrow screens", Type: "Bug", Priority: "Medium", Assignee: "Ada Lovelace"},
				{Key: "DASH-25", Summary: "Document how to create a Jira API token", Type: "Task", Priority: "Low"},
			}},
			{Name: "In Progress", Issues: []Issue{
				{Key: "DASH-18", Summary: "Fetch board columns and their statuses from the Jira Agile API", Type: "Story", Priority: "Highest", Assignee: "Grace Hopper"},
				{Key: "DASH-19", Summary: "Load configuration from .env at startup", Type: "Task", Priority: "Medium", Assignee: "Alan Turing"},
			}},
			{Name: "In Review", Issues: []Issue{
				{Key: "DASH-15", Summary: "Render a read-only Kanban board from mock data, so the layout and styling can be tried out before the app connects to Jira", Type: "Story", Priority: "Medium", Assignee: "Grace Hopper"},
			}},
			{Name: "Blocked"},
			{Name: "Done", Issues: []Issue{
				{Key: "DASH-12", Summary: "Set up templ and Tailwind tooling", Type: "Task", Assignee: "Ada Lovelace"},
				{Key: "DASH-11", Summary: "Initialize the project from the Go template", Type: "Task", Priority: "Lowest", Assignee: "Alan Turing"},
			}},
		},
	}, nil
}
