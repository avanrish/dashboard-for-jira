// Package config loads the dashboard's settings from the environment, the settings file, and the system's password store.
package config

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/zalando/go-keyring"
)

// AppName names the settings directory and the password store entry, and is the command users run.
const AppName = "dashboard-for-jira"

// Config holds the dashboard's settings; the README describes each one.
type Config struct {
	Port         int
	JiraSite     string
	JiraEmail    string
	JiraAPIToken string
	JiraBoardID  int
}

// SettingsError lists every problem with the settings.
type SettingsError struct {
	// Missing names the settings that are set nowhere, by their environment variables, such as JIRA_SITE.
	// It includes JIRA_API_TOKEN when JIRA_EMAIL is missing, because the saved token is found by email address.
	Missing  []string
	Problems []string // every problem, the missing settings among them
}

func (e *SettingsError) Error() string {
	return fmt.Sprintf("invalid settings: %s", strings.Join(e.Problems, "; "))
}

// File holds the settings kept in the settings file.
// It has no field for the API token, so a file that holds one fails to load.
type File struct {
	Site    string `json:"site"`
	Email   string `json:"email"`
	BoardID int    `json:"boardId"`
	Port    int    `json:"port,omitzero"` // 0 means the default
}

// Path returns where the settings file is: config.json in the app's directory under the user's config directory.
func Path() (string, error) {
	dir, err := userConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, AppName, "config.json"), nil
}

// userConfigDir is os.UserConfigDir, except that macOS uses $XDG_CONFIG_HOME or ~/.config, like Linux,
// where command-line tools keep their settings, instead of ~/Library/Application Support.
func userConfigDir() (string, error) {
	if runtime.GOOS != "darwin" {
		return os.UserConfigDir()
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(dir) {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config"), nil
}

// ReadFile reads the settings file at path. A missing or empty file gives empty settings rather than an error.
// Anything but a single JSON object with known, unrepeated keys is an error, so a mistake in a hand-edited file is reported instead of ignored.
func ReadFile(path string) (File, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if errors.Is(err, fs.ErrNotExist) {
		return File{}, nil
	}
	if err != nil {
		return File{}, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return File{}, nil
	}

	// Unlike encoding/json's Decoder, which stops after the first value, json/v2 also rejects anything after the object.
	var file File
	if err := json.Unmarshal(data, &file, json.RejectUnknownMembers(true)); err != nil {
		return File{}, fmt.Errorf("%s: %w", path, err)
	}
	return file, nil
}

// Save writes the settings file at path, creating its directory, readable only by the user.
func (f File) Save(path string) error {
	data, err := json.Marshal(f, jsontext.WithIndent("  "))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return err
	}
	// WriteFile sets the permissions only when it creates the file, and the user may have created it by hand.
	return os.Chmod(path, 0o600)
}

// Load reads settings from environment variables and from the settings file, if it exists.
// Variables set in the environment take precedence over the file.
// The API token comes from JIRA_API_TOKEN, or else from the system's password store.
// When settings are missing or invalid, the error wraps a *SettingsError.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}
	return load(path)
}

func load(path string) (Config, error) {
	file, err := ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("%w\nFix the file, or run %s configure to write it again", err, AppName)
	}
	settings, err := Resolve(file, "")
	if err != nil {
		return Config{}, fmt.Errorf("%w\nRun %s configure to save your settings to %s, or set them in the environment", err, AppName, path)
	}
	return settings, nil
}

// Resolve combines the settings in file with environment variables, which take precedence when they are not empty.
// The API token comes from JIRA_API_TOKEN, else from token when it is not empty, else from the system's password store.
func Resolve(file File, token string) (Config, error) {
	fileValues := map[string]string{
		"JIRA_SITE":     file.Site,
		"JIRA_EMAIL":    file.Email,
		"JIRA_BOARD_ID": FormatNumber(file.BoardID),
		"PORT":          FormatNumber(file.Port),
	}
	return parse(func(name string) string {
		if value := os.Getenv(name); value != "" {
			return value
		}
		return fileValues[name]
	}, func(email string) (string, error) {
		if token != "" {
			return token, nil
		}
		return SavedToken(email)
	})
}

// FormatNumber formats a number from the settings file, where 0 means the setting is missing.
func FormatNumber(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// SavedToken returns the API token saved for email in the system's password store,
// or an error wrapping keyring.ErrNotFound when there is none.
func SavedToken(email string) (string, error) {
	return keyring.Get(AppName, email)
}

// SaveToken saves the API token for email in the system's password store, replacing any saved before.
func SaveToken(email, token string) error {
	return keyring.Set(AppName, email, token)
}

// DeleteToken removes the API token saved for email from the system's password store, if there is one.
func DeleteToken(email string) error {
	if err := keyring.Delete(AppName, email); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	return nil
}

// parse builds the settings from named values, which use the environment variables' names, reporting every problem at once.
func parse(lookup func(name string) string, savedToken func(email string) (string, error)) (Config, error) {
	var missing, problems []string
	required := func(name string) string {
		value := lookup(name)
		if value == "" {
			missing = append(missing, name)
			problems = append(problems, fmt.Sprintf("%s is not set", name))
		}
		return value
	}
	number := func(name, value string, maxValue int) int {
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > maxValue {
			problems = append(problems, fmt.Sprintf("%s must be a whole number from 1 to %d, not %q", name, maxValue, value))
		}
		return n
	}

	settings := Config{
		JiraSite:     required("JIRA_SITE"),
		JiraEmail:    required("JIRA_EMAIL"),
		JiraAPIToken: lookup("JIRA_API_TOKEN"),
	}
	if strings.ContainsAny(settings.JiraSite, "/:") {
		problems = append(problems, "JIRA_SITE must be a host name such as example.atlassian.net, without https://")
	}
	if settings.JiraAPIToken == "" && settings.JiraEmail == "" {
		missing = append(missing, "JIRA_API_TOKEN")
	}
	if settings.JiraAPIToken == "" && settings.JiraEmail != "" {
		token, err := savedToken(settings.JiraEmail)
		switch {
		case errors.Is(err, keyring.ErrNotFound):
			missing = append(missing, "JIRA_API_TOKEN")
			problems = append(problems, fmt.Sprintf("the password store has no API token for %s, and JIRA_API_TOKEN is not set", settings.JiraEmail))
		case err != nil:
			problems = append(problems, fmt.Sprintf("reading the API token from the password store failed (%v), and JIRA_API_TOKEN is not set", err))
		}
		settings.JiraAPIToken = token
	}
	if boardID := required("JIRA_BOARD_ID"); boardID != "" {
		settings.JiraBoardID = number("JIRA_BOARD_ID", boardID, math.MaxInt)
	}
	port := lookup("PORT")
	if port == "" {
		port = "8080"
	}
	settings.Port = number("PORT", port, 65535)

	if len(problems) > 0 {
		return Config{}, &SettingsError{Missing: missing, Problems: problems}
	}
	return settings, nil
}
