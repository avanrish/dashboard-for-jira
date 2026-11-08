// Package config loads the dashboard's settings from the environment and an optional .env file.
package config

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/avanrish/dashboard-for-jira/internal/dotenv"
)

// Config holds the dashboard's settings; .env.example describes each one.
type Config struct {
	Port         int
	JiraSite     string
	JiraEmail    string
	JiraAPIToken string
	JiraBoardID  int
}

// Load reads settings from environment variables and from .env in the current directory, if it exists.
// Variables set in the environment take precedence over the file.
func Load() (Config, error) {
	fileValues, err := dotenv.Read(".env")
	if err != nil {
		return Config{}, err
	}
	return parse(func(name string) string {
		if value, ok := os.LookupEnv(name); ok {
			return value
		}
		return fileValues[name]
	})
}

func parse(lookup func(name string) string) (Config, error) {
	var problems []string
	required := func(name string) string {
		value := lookup(name)
		if value == "" {
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
		JiraAPIToken: required("JIRA_API_TOKEN"),
	}
	if strings.ContainsAny(settings.JiraSite, "/:") {
		problems = append(problems, "JIRA_SITE must be a host name such as example.atlassian.net, without https://")
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
		return Config{}, fmt.Errorf("invalid settings, see .env.example: %s", strings.Join(problems, "; "))
	}
	return settings, nil
}
