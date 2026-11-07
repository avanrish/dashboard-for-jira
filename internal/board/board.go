// Package board models a Kanban board independently of where its data comes from.
package board

type Board struct {
	Name    string
	Columns []Column
}

type Column struct {
	Name   string
	Issues []Issue
}

type Issue struct {
	Key      string
	Summary  string
	Type     string // e.g. Story, Bug, or Task
	Priority string // empty when unset
	Assignee string // display name; empty when unassigned
}
