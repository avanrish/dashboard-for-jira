// Package board models a Kanban board independently of where its data comes from.
package board

type Board struct {
	Name    string
	Columns []string // column names, left to right
	Lanes   []Lane   // one per epic, then one for issues without an epic
}

// Lane holds the issues of one epic, split by column.
type Lane struct {
	Epic  Epic      // zero for the lane of issues without an epic
	Cells [][]Issue // indexed like Board.Columns
}

type Epic struct {
	Key     string
	Summary string
	URL     string
	Color   string // one of Jira's epic colours, such as "purple" or "dark-teal"; empty when unset
}

type Issue struct {
	Key            string
	Summary        string
	Type           string // e.g. Story, Bug, or Task
	Status         string
	StatusCategory StatusCategory
	Priority       string // empty when unset
	Assignee       string // display name; empty when unassigned
	AssigneeAvatar string // URL of the assignee's avatar picture from Jira
	Team           string // empty when unset
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

func (l Lane) IssueCount() int {
	count := 0
	for _, issues := range l.Cells {
		count += len(issues)
	}
	return count
}

func (b Board) ColumnIssueCount(column int) int {
	count := 0
	for _, lane := range b.Lanes {
		count += len(lane.Cells[column])
	}
	return count
}
