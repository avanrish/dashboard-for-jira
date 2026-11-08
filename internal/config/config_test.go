package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadPrefersEnvironmentOverFile(t *testing.T) {
	t.Chdir(t.TempDir())
	content := "JIRA_SITE=file.atlassian.net\nJIRA_EMAIL=ada@example.com\nJIRA_API_TOKEN=token\nJIRA_BOARD_ID=7\n"
	if err := os.WriteFile(".env", []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"JIRA_EMAIL", "JIRA_API_TOKEN", "JIRA_BOARD_ID"} {
		t.Setenv(name, "") // restores the variable after the test
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("JIRA_SITE", "env.atlassian.net")
	t.Setenv("PORT", "9090")

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := Config{Port: 9090, JiraSite: "env.atlassian.net", JiraEmail: "ada@example.com", JiraAPIToken: "token", JiraBoardID: 7}
	if got != want {
		t.Errorf("Load() = %+v, want %+v", got, want)
	}
}

func TestParseReportsEveryProblem(t *testing.T) {
	values := map[string]string{"JIRA_SITE": "https://example.atlassian.net", "JIRA_BOARD_ID": "seven", "PORT": "70000"}
	_, err := parse(func(name string) string { return values[name] })
	if err == nil {
		t.Fatal("parse() succeeded, want an error")
	}
	for _, want := range []string{"JIRA_SITE must be a host name", "JIRA_EMAIL is not set", "JIRA_API_TOKEN is not set", "JIRA_BOARD_ID must be", "PORT must be"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("parse() error = %q, want it to contain %q", err, want)
		}
	}
}
