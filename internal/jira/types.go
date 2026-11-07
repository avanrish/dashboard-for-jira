package jira

import "encoding/json"

type BoardConfiguration struct {
	Name         string `json:"name"`
	Type         string `json:"type"` // kanban or scrum
	ColumnConfig struct {
		Columns []BoardColumn `json:"columns"`
	} `json:"columnConfig"`
	// SubQuery is a Kanban board's extra filter, by default one that hides issues in released versions.
	SubQuery struct {
		Query string `json:"query"`
	} `json:"subQuery"`
}

type BoardColumn struct {
	Name     string `json:"name"`
	Statuses []struct {
		ID string `json:"id"`
	} `json:"statuses"`
}

type Issue struct {
	ID     string      `json:"id"`
	Key    string      `json:"key"`
	Fields IssueFields `json:"fields"`
}

type IssueFields struct {
	Summary string `json:"summary"`
	Status  struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		StatusCategory struct {
			Key string `json:"key"` // new, indeterminate, or done
		} `json:"statusCategory"`
	} `json:"status"`
	IssueType IssueType `json:"issuetype"`
	Priority  *struct {
		Name string `json:"name"`
	} `json:"priority"`
	Assignee *struct {
		DisplayName string            `json:"displayName"`
		AvatarURLs  map[string]string `json:"avatarUrls"` // by size, such as 48x48
	} `json:"assignee"`
	Parent *Parent `json:"parent"`
	// All holds every returned field undecoded, for custom fields whose IDs differ between sites, such as Team.
	All map[string]json.RawMessage `json:"-"`
}

func (f *IssueFields) UnmarshalJSON(data []byte) error {
	type namedFields IssueFields // has no UnmarshalJSON method, so decoding it does not recurse
	if err := json.Unmarshal(data, (*namedFields)(f)); err != nil {
		return err
	}
	return json.Unmarshal(data, &f.All)
}

type IssueType struct {
	Name           string `json:"name"`
	HierarchyLevel int    `json:"hierarchyLevel"`
}

type Parent struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Fields struct {
		Summary   string    `json:"summary"`
		IssueType IssueType `json:"issuetype"`
	} `json:"fields"`
}

// CustomFields holds the IDs of the custom fields the board shows; an ID is "" when the site has no such field.
type CustomFields struct {
	Team       string
	IssueColor string
}
