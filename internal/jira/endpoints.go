package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Types of custom fields the board shows. Their IDs differ on every site.
const (
	teamFieldType       = "com.atlassian.teams:rm-teams-custom-field-team"
	issueColorFieldType = "com.pyxis.greenhopper.jira:jsw-issue-color" // an epic's colour
)

// Myself returns the display name of the user the client signs in as, which checks the email address and API token.
// It always asks Jira, so a token revoked since an earlier call fails the check.
func (c *Client) Myself(ctx context.Context) (string, error) {
	var user struct {
		DisplayName string `json:"displayName"`
	}
	err := c.get(ctx, "/rest/api/3/myself", nil, &user, RequestOptions{})
	return user.DisplayName, err
}

func (c *Client) BoardConfiguration(ctx context.Context, boardID int, options RequestOptions) (BoardConfiguration, error) {
	var configuration BoardConfiguration
	err := c.get(ctx, fmt.Sprintf("/rest/agile/1.0/board/%d/configuration", boardID), nil, &configuration, options)
	return configuration, err
}

func (c *Client) CustomFields(ctx context.Context, options RequestOptions) (CustomFields, error) {
	var fields []struct {
		ID     string `json:"id"`
		Schema struct {
			Custom string `json:"custom"`
		} `json:"schema"`
	}
	if err := c.get(ctx, "/rest/api/3/field", nil, &fields, options); err != nil {
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
func (c *Client) BoardIssues(ctx context.Context, boardID int, jql string, fields []string, options RequestOptions) ([]Issue, error) {
	query := url.Values{"jql": {jql}, "fields": {strings.Join(fields, ",")}}
	return c.allIssues(ctx, fmt.Sprintf("/rest/software/1.0/board/%d/issue", boardID), query, options)
}

// SearchIssues returns the issues that match jql, in the query's order, following every page.
// Without fields, Jira returns only each issue's ID.
func (c *Client) SearchIssues(ctx context.Context, jql string, fields []string, options RequestOptions) ([]Issue, error) {
	query := url.Values{"jql": {jql}}
	if len(fields) > 0 {
		query.Set("fields", strings.Join(fields, ","))
	}
	return c.allIssues(ctx, "/rest/api/3/search/jql", query, options)
}

// allIssues returns the issues from every page of an endpoint that pages them with nextPageToken.
// It caches the pages as one response, so a result never mixes pages that Jira sent at different times.
func (c *Client) allIssues(ctx context.Context, path string, query url.Values, options RequestOptions) ([]Issue, error) {
	// The most Jira allows; it sends smaller pages when it needs to.
	query.Set("maxResults", "5000")
	endpoint := c.baseURL.JoinPath(path)
	endpoint.RawQuery = query.Encode()

	var issues []Issue
	err := c.cached(endpoint.String(), options, &issues, func() ([]byte, error) {
		var all []json.RawMessage
		for {
			var page struct {
				Issues        []json.RawMessage `json:"issues"`
				NextPageToken string            `json:"nextPageToken"`
			}
			body, err := c.fetch(ctx, endpoint.String())
			if err != nil {
				return nil, err
			}
			if err := json.Unmarshal(body, &page); err != nil {
				return nil, err
			}
			all = append(all, page.Issues...)
			if page.NextPageToken == "" {
				return json.Marshal(all)
			}
			query.Set("nextPageToken", page.NextPageToken)
			endpoint.RawQuery = query.Encode()
		}
	})
	if err != nil {
		return nil, err
	}
	return issues, nil
}
