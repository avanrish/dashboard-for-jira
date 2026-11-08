package dotenv

import (
	"maps"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	content := `# Jira settings

JIRA_SITE=example.atlassian.net
export JIRA_EMAIL = "ada@example.com"
JIRA_API_TOKEN='abc=def'
PORT=
GREETING="  hi  "
`
	got, err := Parse(strings.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"JIRA_SITE":      "example.atlassian.net",
		"JIRA_EMAIL":     "ada@example.com",
		"JIRA_API_TOKEN": "abc=def",
		"PORT":           "",
		"GREETING":       "  hi  ",
	}
	if !maps.Equal(got, want) {
		t.Errorf("Parse() = %v, want %v", got, want)
	}

	if _, err := Parse(strings.NewReader("JIRA_SITE=example.atlassian.net\nnot a setting\n")); err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Errorf("Parse() error = %v, want one naming line 2", err)
	}
}

// The input is a quoted string so that its trailing whitespace is visible and no editor strips it.
func TestParseTrimsUnquotedValues(t *testing.T) {
	got, err := Parse(strings.NewReader("MY_VAR=123 \t \n"))
	if err != nil || got["MY_VAR"] != "123" {
		t.Errorf("Parse() = %v, %v, want MY_VAR=123", got, err)
	}
}

func TestReadMissingFile(t *testing.T) {
	got, err := Read(filepath.Join(t.TempDir(), ".env"))
	if err != nil || len(got) != 0 {
		t.Errorf("Read() = %v, %v, want no values and no error", got, err)
	}
}
