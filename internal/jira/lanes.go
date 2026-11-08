package jira

import (
	"cmp"
	"encoding/json"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/avanrish/dashboard-for-jira/internal/board"
)

// epicLevel is the IssueType.HierarchyLevel of epics; standard issues are 0 and subtasks -1.
const epicLevel = 1

// epicColors lists Jira's epic colours in the order of their codes, ghx-label-1 to ghx-label-14.
var epicColors = []string{
	"dark-gray", "dark-yellow", "yellow", "dark-blue", "dark-teal", "green", "purple",
	"dark-purple", "orange", "blue", "teal", "gray", "dark-green", "dark-orange",
}

// missingParents returns the IDs of subtasks' parents that are not among issues, such as stories in the backlog.
// epicsByIssue needs them to find the subtasks' epics.
func missingParents(issues []Issue) []string {
	keys := make(map[string]bool)
	for _, issue := range issues {
		keys[issue.Key] = true
	}
	ids := make(map[string]bool)
	for _, issue := range issues {
		if parent := issue.Fields.Parent; parent != nil && parent.Fields.IssueType.HierarchyLevel < epicLevel && !keys[parent.Key] {
			ids[parent.ID] = true
		}
	}
	return slices.Sorted(maps.Keys(ids))
}

// epicsByIssue maps issue keys to their epics. A subtask's parent is a standard issue, so a subtask takes that issue's epic.
// parents holds the parents that missingParents reports, with their parent field.
func epicsByIssue(issues, parents []Issue) map[string]*Parent {
	epics := make(map[string]*Parent)
	for _, issue := range slices.Concat(issues, parents) {
		if parent := issue.Fields.Parent; parent != nil && parent.Fields.IssueType.HierarchyLevel == epicLevel {
			epics[issue.Key] = parent
		}
	}
	for _, issue := range issues {
		if parent := issue.Fields.Parent; parent != nil && epics[issue.Key] == nil && epics[parent.Key] != nil {
			epics[issue.Key] = epics[parent.Key]
		}
	}
	return epics
}

// buildBoard puts each issue in its status's column and its epic's lane.
// Only epics with issues get a lane, ordered like rankedEpics, which also holds their colours.
func buildBoard(
	name string, columns []BoardColumn, issues []Issue, epics map[string]*Parent, rankedEpics []Issue,
	customFields CustomFields, issueURL func(key string) string,
) board.Board {
	result := board.Board{Name: name}
	columnOfStatus := make(map[string]int)
	for i, column := range columns {
		result.Columns = append(result.Columns, column.Name)
		for _, status := range column.Statuses {
			columnOfStatus[status.ID] = i
		}
	}

	positionOfEpic := make(map[string]int)
	colorOfEpic := make(map[string]string)
	for i, epic := range rankedEpics {
		positionOfEpic[epic.ID] = i
		colorOfEpic[epic.ID] = epicColor(epic.Fields.All[customFields.IssueColor])
	}

	laneOfEpic := make(map[string]*board.Lane)
	var epicIDs []string
	withoutEpic := &board.Lane{Cells: make([][]board.Issue, len(columns))}
	for _, issue := range issues {
		column, onBoard := columnOfStatus[issue.Fields.Status.ID]
		if !onBoard || issue.Fields.IssueType.HierarchyLevel == epicLevel {
			continue
		}
		lane := withoutEpic
		if epic := epics[issue.Key]; epic != nil {
			lane = laneOfEpic[epic.ID]
			if lane == nil {
				lane = &board.Lane{
					Epic: board.Epic{
						Key: epic.Key, Summary: epic.Fields.Summary, URL: issueURL(epic.Key),
						Color: colorOfEpic[epic.ID],
					},
					Cells: make([][]board.Issue, len(columns)),
				}
				laneOfEpic[epic.ID] = lane
				epicIDs = append(epicIDs, epic.ID)
			}
		}
		lane.Cells[column] = append(lane.Cells[column], cardOf(issue, customFields.Team, issueURL))
	}

	position := func(epicID string) int {
		if i, ok := positionOfEpic[epicID]; ok {
			return i
		}
		return len(rankedEpics)
	}
	slices.SortStableFunc(epicIDs, func(a, b string) int { return cmp.Compare(position(a), position(b)) })
	for _, id := range epicIDs {
		result.Lanes = append(result.Lanes, *laneOfEpic[id])
	}
	if withoutEpic.IssueCount() > 0 {
		result.Lanes = append(result.Lanes, *withoutEpic)
	}
	return result
}

// epicColor turns an issue colour value, a code such as ghx-label-7 or a name such as "dark_blue", into one of epicColors.
// It returns "" for no colour or an unknown one.
func epicColor(value json.RawMessage) string {
	var color string
	if value == nil || json.Unmarshal(value, &color) != nil {
		return ""
	}
	color = strings.ToLower(strings.TrimSpace(color))
	if code, ok := strings.CutPrefix(color, "ghx-label-"); ok {
		if n, err := strconv.Atoi(code); err == nil && n >= 1 && n <= len(epicColors) {
			return epicColors[n-1]
		}
		return ""
	}
	color = strings.NewReplacer("_", "-", " ", "-", "grey", "gray").Replace(color)
	if slices.Contains(epicColors, color) {
		return color
	}
	return ""
}

func cardOf(issue Issue, teamField string, issueURL func(key string) string) board.Issue {
	card := board.Issue{
		Key:     issue.Key,
		Summary: issue.Fields.Summary,
		Type:    issue.Fields.IssueType.Name,
		Status:  issue.Fields.Status.Name,
		Flagged: issue.Fields.Flagged,
		URL:     issueURL(issue.Key),
	}
	switch issue.Fields.Status.StatusCategory.Key {
	case "indeterminate":
		card.StatusCategory = board.InProgress
	case "done":
		card.StatusCategory = board.Done
	}
	if issue.Fields.Priority != nil {
		card.Priority = issue.Fields.Priority.Name
	}
	if issue.Fields.Assignee != nil {
		card.Assignee = issue.Fields.Assignee.DisplayName
		card.AssigneeAvatar = issue.Fields.Assignee.AvatarURLs["48x48"]
	}
	if value := issue.Fields.All[teamField]; teamField != "" && value != nil {
		var team struct {
			Name string `json:"name"`
		}
		// A value in another shape leaves the team blank rather than failing the whole board.
		if json.Unmarshal(value, &team) == nil {
			card.Team = team.Name
		}
	}
	return card
}
