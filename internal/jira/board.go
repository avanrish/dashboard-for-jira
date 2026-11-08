package jira

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/avanrish/dashboard-for-jira/internal/board"
)

// boardFields lists the issue fields the board shows; requesting only these keeps responses small.
var boardFields = []string{"summary", "status", "issuetype", "priority", "assignee", "parent", "flagged"}

// recentlyDone hides finished issues not updated for 7 days, as the board in Jira does, which keeps long-lived boards short.
const recentlyDone = "statusCategory != Done OR updated >= -7d"

// BoardSource loads a Jira board with its issues grouped into epic lanes, as the board in Jira shows them.
type BoardSource struct {
	Client  *Client
	BoardID int
}

func (s BoardSource) Board(ctx context.Context) (board.Board, error) {
	var configuration BoardConfiguration
	var customFields CustomFields
	var configurationErr, customFieldsErr error
	var wait sync.WaitGroup
	wait.Go(func() { configuration, configurationErr = s.Client.BoardConfiguration(ctx, s.BoardID) })
	wait.Go(func() { customFields, customFieldsErr = s.Client.CustomFields(ctx) })
	wait.Wait()
	if err := errors.Join(configurationErr, customFieldsErr); err != nil {
		return board.Board{}, fmt.Errorf("loading the settings of board %d: %w", s.BoardID, err)
	}

	columns := configuration.ColumnConfig.Columns
	// With the Kanban backlog on, Jira lists it as a first column named Backlog, which the board does not show.
	// The API has no flag for it: the features endpoint that reports it only works for team-managed boards.
	if configuration.Type == "kanban" && len(columns) > 0 && strings.EqualFold(columns[0].Name, "Backlog") {
		columns = columns[1:]
	}
	var statusIDs []string
	for _, column := range columns {
		for _, status := range column.Statuses {
			statusIDs = append(statusIDs, status.ID)
		}
	}
	if len(statusIDs) == 0 {
		return buildBoard(configuration.Name, columns, nil, nil, nil, customFields, s.Client.IssueURL), nil
	}

	jql := fmt.Sprintf("status in (%s) AND (%s)", strings.Join(statusIDs, ", "), recentlyDone)
	if configuration.SubQuery.Query != "" {
		jql = fmt.Sprintf("(%s) AND %s", configuration.SubQuery.Query, jql)
	}
	fields := boardFields
	if customFields.Team != "" {
		fields = append(slices.Clone(boardFields), customFields.Team)
	}
	issues, err := s.Client.BoardIssues(ctx, s.BoardID, jql, fields)
	if err != nil {
		return board.Board{}, fmt.Errorf("loading the issues of board %d: %w", s.BoardID, err)
	}

	epics := epicsByIssue(issues)
	var rankedEpics []Issue
	if len(epics) > 0 {
		ids := make(map[string]bool)
		for _, epic := range epics {
			ids[epic.ID] = true
		}
		var epicFields []string
		if customFields.IssueColor != "" {
			epicFields = []string{customFields.IssueColor}
		}
		jql := fmt.Sprintf("id in (%s) ORDER BY Rank ASC", strings.Join(slices.Sorted(maps.Keys(ids)), ", "))
		rankedEpics, err = s.Client.SearchIssues(ctx, jql, epicFields)
		if err != nil {
			return board.Board{}, fmt.Errorf("ordering the epics of board %d: %w", s.BoardID, err)
		}
	}
	return buildBoard(configuration.Name, columns, issues, epics, rankedEpics, customFields, s.Client.IssueURL), nil
}
