package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

// unsetenv unsets the variables for the test and restores them afterwards.
func unsetenv(t *testing.T, names ...string) {
	t.Helper()
	for _, name := range names {
		t.Setenv(name, "")
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
}

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadPrefersEnvironmentOverFile(t *testing.T) {
	keyring.MockInit()
	if err := SaveToken("ada@example.com", "saved-token"); err != nil {
		t.Fatal(err)
	}
	path := writeFile(t, `{"site": "file.atlassian.net", "email": "ada@example.com", "boardId": 7}`)
	unsetenv(t, "JIRA_EMAIL", "JIRA_API_TOKEN", "JIRA_BOARD_ID")
	t.Setenv("JIRA_SITE", "env.atlassian.net")
	t.Setenv("PORT", "9090")

	got, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	want := Config{Port: 9090, JiraSite: "env.atlassian.net", JiraEmail: "ada@example.com", JiraAPIToken: "saved-token", JiraBoardID: 7}
	if got != want {
		t.Errorf("load() = %+v, want %+v", got, want)
	}

	t.Setenv("JIRA_API_TOKEN", "env-token")
	if got, err := load(path); err != nil || got.JiraAPIToken != "env-token" {
		t.Errorf("load() = %+v, %v, want the token from JIRA_API_TOKEN", got, err)
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	keyring.MockInit()
	path := writeFile(t, `{"site": "example.atlassian.net", "email": "ada@example.com", "apiToken": "secret", "boardId": 7}`)
	unsetenv(t, "JIRA_SITE", "JIRA_EMAIL", "JIRA_API_TOKEN", "JIRA_BOARD_ID", "PORT")

	_, err := load(path)
	if err == nil || !strings.Contains(err.Error(), `unknown field "apiToken"`) {
		t.Errorf("load() error = %v, want one about the unknown apiToken key", err)
	}
}

func TestLoadReportsMissingSettings(t *testing.T) {
	keyring.MockInit()
	path := filepath.Join(t.TempDir(), "config.json")
	unsetenv(t, "JIRA_SITE", "JIRA_EMAIL", "JIRA_API_TOKEN", "JIRA_BOARD_ID", "PORT")

	_, err := load(path)
	if err == nil || !strings.Contains(err.Error(), "dashboard-for-jira configure") || !strings.Contains(err.Error(), path) {
		t.Errorf("load() error = %v, want it to suggest configure and name %s", err, path)
	}
	var settingsErr *SettingsError
	want := []string{"JIRA_SITE", "JIRA_EMAIL", "JIRA_API_TOKEN", "JIRA_BOARD_ID"}
	if !errors.As(err, &settingsErr) || !slices.Equal(settingsErr.Missing, want) {
		t.Errorf("load() error = %#v, want a *SettingsError missing %v", err, want)
	}
}

func TestResolveReportsOnlyTheMissingToken(t *testing.T) {
	keyring.MockInit()
	unsetenv(t, "JIRA_SITE", "JIRA_EMAIL", "JIRA_API_TOKEN", "JIRA_BOARD_ID", "PORT")
	file := File{Site: "example.atlassian.net", Email: "ada@example.com", BoardID: 7}

	_, err := Resolve(file, "")
	var settingsErr *SettingsError
	if !errors.As(err, &settingsErr) || !slices.Equal(settingsErr.Missing, []string{"JIRA_API_TOKEN"}) {
		t.Errorf("Resolve() error = %#v, want a *SettingsError missing only JIRA_API_TOKEN", err)
	}
	if got, err := Resolve(file, "new-token"); err != nil || got.JiraAPIToken != "new-token" {
		t.Errorf("Resolve() with a token = %+v, %v, want it to use the token", got, err)
	}
}

func TestParseReportsEveryProblem(t *testing.T) {
	values := map[string]string{"JIRA_SITE": "https://example.atlassian.net", "JIRA_EMAIL": "ada@example.com", "JIRA_BOARD_ID": "seven", "PORT": "70000"}
	noToken := func(string) (string, error) { return "", keyring.ErrNotFound }
	_, err := parse(func(name string) string { return values[name] }, noToken)
	if err == nil {
		t.Fatal("parse() succeeded, want an error")
	}
	for _, want := range []string{"JIRA_SITE must be a host name", "no API token for ada@example.com", "JIRA_BOARD_ID must be", "PORT must be"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("parse() error = %q, want it to contain %q", err, want)
		}
	}
}

func TestParseReportsUnavailablePasswordStore(t *testing.T) {
	values := map[string]string{"JIRA_SITE": "example.atlassian.net", "JIRA_EMAIL": "ada@example.com", "JIRA_BOARD_ID": "7"}
	unavailable := func(string) (string, error) { return "", errors.New("no secret service") }
	_, err := parse(func(name string) string { return values[name] }, unavailable)
	if err == nil || !strings.Contains(err.Error(), "no secret service") || !strings.Contains(err.Error(), "JIRA_API_TOKEN") {
		t.Errorf("parse() error = %v, want the password store's error and a hint about JIRA_API_TOKEN", err)
	}
}

func TestPathFollowsXDGConfigHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows keeps settings under %AppData%")
	}
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	got, err := Path()
	if want := filepath.Join(dir, "dashboard-for-jira", "config.json"); err != nil || got != want {
		t.Errorf("Path() = %q, %v, want %q", got, err, want)
	}
}

func TestSaveWritesAFileOnlyTheUserCanRead(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dashboard-for-jira", "config.json")
	file := File{Site: "example.atlassian.net", Email: "ada@example.com", BoardID: 7}
	if err := file.Save(path); err != nil {
		t.Fatal(err)
	}

	if got, err := ReadFile(path); err != nil || got != file {
		t.Errorf("ReadFile() = %+v, %v, want %+v", got, err, file)
	}
	if runtime.GOOS == "windows" {
		return // Windows has no Unix permission bits.
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("file mode = %v, want -rw-------", mode)
	}
}

func TestReadMissingFile(t *testing.T) {
	got, err := ReadFile(filepath.Join(t.TempDir(), "config.json"))
	if err != nil || got != (File{}) {
		t.Errorf("ReadFile() = %+v, %v, want empty settings and no error", got, err)
	}
}
