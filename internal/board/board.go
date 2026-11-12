// Package board models a Kanban board independently of where its data comes from.
package board

type Board struct {
	Name    string
	Columns []string
	Epics   []Epic // one per epic, then one without a Key for issues that have no epic
}

// Epic holds the issues of one epic, split by column.
type Epic struct {
	Key     string
	Summary string
	URL     string
	Color   string    // one of Jira's epic colours, such as "purple" or "dark-teal"; empty when unset
	Cells   [][]Issue // indexed like Board.Columns
}

type Issue struct {
	Key            string
	Summary        string
	Type           string
	Status         string
	StatusCategory StatusCategory
	Priority       string
	Assignee       string // display name
	AssigneeAvatar string
	Team           string
	Flagged        bool
	URL            string
}

// StatusCategory groups statuses by how far along an issue is.
type StatusCategory int

const (
	ToDo StatusCategory = iota
	InProgress
	Done
)

func (e Epic) IssueCount() int {
	count := 0
	for _, issues := range e.Cells {
		count += len(issues)
	}
	return count
}

func (b Board) ColumnIssueCount(column int) int {
	count := 0
	for _, epic := range b.Epics {
		count += len(epic.Cells[column])
	}
	return count
}
