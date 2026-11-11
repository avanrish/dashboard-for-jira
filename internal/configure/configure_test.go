package configure

import (
	"bufio"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"

	"github.com/avanrish/dashboard-for-jira/internal/config"
)

func everything(string) bool { return true }

// newTestSession returns a session that answers the questions with input and talks to a fake Jira that accepts token.
func newTestSession(t *testing.T, input, token string) (session, *strings.Builder) {
	t.Helper()
	keyring.MockInit()
	for _, name := range []string{"JIRA_SITE", "JIRA_EMAIL", "JIRA_API_TOKEN", "JIRA_BOARD_ID", "PORT"} {
		t.Setenv(name, "")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if email, got, _ := r.BasicAuth(); email == "" || got != token {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"errorMessages": ["Client must be authenticated to access this resource."]}`)
			return
		}
		switch r.URL.Path {
		case "/rest/api/3/myself":
			_, _ = io.WriteString(w, `{"displayName": "Ada Lovelace"}`)
		case "/rest/agile/1.0/board/12/configuration":
			_, _ = io.WriteString(w, `{"name": "Team board"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	in := bufio.NewReader(strings.NewReader(input))
	out := new(strings.Builder)
	return session{
		path:       filepath.Join(t.TempDir(), "dashboard-for-jira", "config.json"),
		in:         in,
		out:        out,
		readSecret: func() (string, error) { return readLine(in) },
		siteURL:    func(string) *url.URL { return serverURL },
	}, out
}

func checkFile(t *testing.T, path string, want config.File) {
	t.Helper()
	got, err := config.ReadFile(path)
	if err != nil || got != want {
		t.Errorf("settings file = %+v, %v, want %+v", got, err, want)
	}
}

func checkSavedToken(t *testing.T, email, want string) {
	t.Helper()
	got, err := config.SavedToken(email)
	if want == "" && !errors.Is(err, keyring.ErrNotFound) {
		t.Errorf("token saved for %s = %q, %v, want none", email, got, err)
	}
	if want != "" && (err != nil || got != want) {
		t.Errorf("token saved for %s = %q, %v, want %q", email, got, err, want)
	}
}

func TestRunSavesSettings(t *testing.T) {
	boardAddress := "https://example.atlassian.net/jira/software/projects/ABC/boards/12"
	input := strings.Join([]string{boardAddress, "ada", "ada@example.com", "secret", "twelve", boardAddress, "0", ""}, "\n") + "\n"
	s, out := newTestSession(t, input, "secret")

	settings, err := s.run(t.Context(), everything)
	if err != nil {
		t.Fatalf("run() error = %v, output:\n%s", err, out)
	}
	want := config.Config{Port: 8080, JiraSite: "example.atlassian.net", JiraEmail: "ada@example.com", JiraAPIToken: "secret", JiraBoardID: 12}
	if settings != want {
		t.Errorf("run() = %+v, want %+v", settings, want)
	}
	checkFile(t, s.path, config.File{Site: "example.atlassian.net", Email: "ada@example.com", BoardID: 12, Port: 8080})
	checkSavedToken(t, "ada@example.com", "secret")
	for _, want := range []string{"enter the email address", "enter the board's ID", "enter a port number", `Signed in as Ada Lovelace. Found board "Team board".`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
}

func TestRunKeepsCurrentValues(t *testing.T) {
	s, out := newTestSession(t, "\n\n\n\n\n", "saved-token")
	current := config.File{Site: "example.atlassian.net", Email: "ada@example.com", BoardID: 12, Port: 9090}
	if err := current.Save(s.path); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveToken("ada@example.com", "saved-token"); err != nil {
		t.Fatal(err)
	}

	if _, err := s.run(t.Context(), everything); err != nil {
		t.Fatalf("run() error = %v, output:\n%s", err, out)
	}
	checkFile(t, s.path, current)
	checkSavedToken(t, "ada@example.com", "saved-token")
}

func TestRunMovesTokenToNewEmail(t *testing.T) {
	s, out := newTestSession(t, "\nbob@example.com\nnew-token\n\n\n", "new-token")
	current := config.File{Site: "example.atlassian.net", Email: "ada@example.com", BoardID: 12}
	if err := current.Save(s.path); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveToken("ada@example.com", "old-token"); err != nil {
		t.Fatal(err)
	}

	if _, err := s.run(t.Context(), everything); err != nil {
		t.Fatalf("run() error = %v, output:\n%s", err, out)
	}
	checkSavedToken(t, "bob@example.com", "new-token")
	checkSavedToken(t, "ada@example.com", "")
}

func TestRunKeepsFileEmailTokenWhenEnvironmentSetsAnother(t *testing.T) {
	s, out := newTestSession(t, "bob-token\n", "bob-token")
	current := config.File{Site: "example.atlassian.net", Email: "ada@example.com", BoardID: 12}
	if err := current.Save(s.path); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveToken("ada@example.com", "ada-token"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JIRA_EMAIL", "bob@example.com")

	missing := []string{"JIRA_API_TOKEN"}
	if _, err := s.run(t.Context(), func(name string) bool { return slices.Contains(missing, name) }); err != nil {
		t.Fatalf("run() error = %v, output:\n%s", err, out)
	}
	checkFile(t, s.path, current)
	checkSavedToken(t, "bob@example.com", "bob-token")
	checkSavedToken(t, "ada@example.com", "ada-token")
}

func TestRunReplacesUnreadableFile(t *testing.T) {
	s, out := newTestSession(t, "example.atlassian.net\nada@example.com\nsecret\n12\n\n", "secret")
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.path, []byte(`{"site": "example.atlassian.net",}`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := s.run(t.Context(), everything); err != nil {
		t.Fatalf("run() error = %v, output:\n%s", err, out)
	}
	if !strings.Contains(out.String(), "cannot be read") {
		t.Errorf("output = %q, want it to say the current settings cannot be read", out)
	}
	checkFile(t, s.path, config.File{Site: "example.atlassian.net", Email: "ada@example.com", BoardID: 12, Port: 8080})
}

func TestRunUsesEnvironment(t *testing.T) {
	s, out := newTestSession(t, "example.atlassian.net\nada@example.com\n12\n\n", "env-token")
	t.Setenv("JIRA_API_TOKEN", "env-token")
	t.Setenv("PORT", "9090")

	settings, err := s.run(t.Context(), everything)
	if err != nil {
		t.Fatalf("run() error = %v, output:\n%s", err, out)
	}
	if settings.JiraAPIToken != "env-token" || settings.Port != 9090 {
		t.Errorf("run() = %+v, want the token and port from the environment", settings)
	}
	if !strings.Contains(out.String(), "PORT is set in the environment") {
		t.Errorf("output = %q, want a note that PORT takes precedence", out)
	}
	checkSavedToken(t, "ada@example.com", "")
}

func TestRunSavesNothingWhenJiraRejectsToken(t *testing.T) {
	s, _ := newTestSession(t, "example.atlassian.net\nada@example.com\nwrong-token\n12\n\n", "secret")

	_, err := s.run(t.Context(), everything)
	if err == nil || !strings.Contains(err.Error(), "nothing was saved") {
		t.Errorf("run() error = %v, want one saying nothing was saved", err)
	}
	if _, err := os.Stat(s.path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("settings file: %v, want it not written", err)
	}
	checkSavedToken(t, "ada@example.com", "")
}

func TestRunStopsWhenInputEnds(t *testing.T) {
	s, _ := newTestSession(t, "example.atlassian.net\n", "secret")
	if _, err := s.run(t.Context(), everything); err == nil || !strings.Contains(err.Error(), "input ended") {
		t.Errorf("run() error = %v, want one about the input ending", err)
	}
}

func TestRunAsksOnlyForMissingSettings(t *testing.T) {
	s, out := newTestSession(t, "secret\n", "secret")
	current := config.File{Site: "example.atlassian.net", Email: "ada@example.com", BoardID: 12}
	if err := current.Save(s.path); err != nil {
		t.Fatal(err)
	}

	missing := []string{"JIRA_API_TOKEN"}
	settings, err := s.run(t.Context(), func(name string) bool { return slices.Contains(missing, name) })
	if err != nil {
		t.Fatalf("run() error = %v, output:\n%s", err, out)
	}
	if settings.JiraAPIToken != "secret" || settings.JiraBoardID != 12 {
		t.Errorf("run() = %+v, want the saved settings with the new token", settings)
	}
	if strings.Contains(out.String(), "Jira site") || strings.Contains(out.String(), "Port") {
		t.Errorf("output = %q, want only the token asked for", out)
	}
	checkFile(t, s.path, current)
	checkSavedToken(t, "ada@example.com", "secret")
}

func TestParseSite(t *testing.T) {
	for answer, want := range map[string]string{
		"example.atlassian.net":                              "example.atlassian.net",
		"example.atlassian.net/jira":                         "example.atlassian.net",
		"https://example.atlassian.net/jira/software/boards": "example.atlassian.net",
		"":                          "",
		"example.atlassian.net:443": "",
	} {
		got, err := parseSite(answer)
		if got != want || (err == nil) != (want != "") {
			t.Errorf("parseSite(%q) = %q, %v, want %q", answer, got, err, want)
		}
	}
}

func TestParseBoardID(t *testing.T) {
	for answer, want := range map[string]int{
		"12": 12,
		"https://example.atlassian.net/jira/software/projects/ABC/boards/12":       12,
		"https://example.atlassian.net/jira/software/c/projects/ABC/boards/12?a=b": 12,
		"0":  0,
		"-3": 0,
		"https://example.atlassian.net/jira/software/projects/ABC/boards": 0,
	} {
		got, err := parseBoardID(answer)
		if got != want || (err == nil) != (want != 0) {
			t.Errorf("parseBoardID(%q) = %d, %v, want %d", answer, got, err, want)
		}
	}
}
