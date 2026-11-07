package jira

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Types of custom fields the board shows. Their IDs differ on every site.
const (
	teamFieldType       = "com.atlassian.teams:rm-teams-custom-field-team"
	issueColorFieldType = "com.pyxis.greenhopper.jira:jsw-issue-color" // an epic's colour
)

func (c *Client) BoardConfiguration(ctx context.Context, boardID int) (BoardConfiguration, error) {
	var configuration BoardConfiguration
	err := c.get(ctx, fmt.Sprintf("/rest/agile/1.0/board/%d/configuration", boardID), nil, &configuration)
	return configuration, err
}

func (c *Client) CustomFields(ctx context.Context) (CustomFields, error) {
	var fields []struct {
		ID     string `json:"id"`
		Schema struct {
			Custom string `json:"custom"`
		} `json:"schema"`
	}
	if err := c.get(ctx, "/rest/api/3/field", nil, &fields); err != nil {
		return CustomFields{}, err
	}
	var found CustomFields
	for _, field := range fields {
		switch {
		case field.Schema.Custom == teamFieldType && found.Team == "":
			found.Team = field.ID
		case field.Schema.Custom == issueColorFieldType && found.IssueColor == "":
			found.IssueColor = field.ID
		}
	}
	return found, nil
}

// BoardIssues returns the board's issues that match jql, in rank order, following every page.
// It uses the Jira Software endpoint because the Agile API's version, paged by offset, is deprecated.
func (c *Client) BoardIssues(ctx context.Context, boardID int, jql string, fields []string) ([]Issue, error) {
	query := url.Values{
		"jql":    {jql},
		"fields": {strings.Join(fields, ",")},
		// The most Jira allows; it sends smaller pages when it needs to.
		"maxResults": {"5000"},
	}
	var issues []Issue
	for {
		var page struct {
			Issues        []Issue `json:"issues"`
			NextPageToken string  `json:"nextPageToken"`
		}
		if err := c.get(ctx, fmt.Sprintf("/rest/software/1.0/board/%d/issue", boardID), query, &page); err != nil {
			return nil, err
		}
		issues = append(issues, page.Issues...)
		if page.NextPageToken == "" {
			return issues, nil
		}
		query.Set("nextPageToken", page.NextPageToken)
	}
}

// SearchIssues returns the issues that match jql, in the query's order, following every page.
// Without fields, Jira returns only each issue's ID.
func (c *Client) SearchIssues(ctx context.Context, jql string, fields []string) ([]Issue, error) {
	query := url.Values{"jql": {jql}, "maxResults": {"5000"}}
	if len(fields) > 0 {
		query.Set("fields", strings.Join(fields, ","))
	}
	var issues []Issue
	for {
		var page struct {
			Issues        []Issue `json:"issues"`
			NextPageToken string  `json:"nextPageToken"`
		}
		if err := c.get(ctx, "/rest/api/3/search/jql", query, &page); err != nil {
			return nil, err
		}
		issues = append(issues, page.Issues...)
		if page.NextPageToken == "" {
			return issues, nil
		}
		query.Set("nextPageToken", page.NextPageToken)
	}
}
