package jira

import (
	"encoding/json"
	"testing"
)

func TestEpicColor(t *testing.T) {
	tests := map[string]string{
		`"ghx-label-1"`:  "dark-gray",
		`"ghx-label-14"`: "dark-orange",
		`"ghx-label-15"`: "",
		`"Dark_Blue"`:    "dark-blue",
		`"dark grey"`:    "dark-gray",
		`"purple"`:       "purple",
		`"red"`:          "",
		`null`:           "",
		`{"value": 1}`:   "",
	}
	for value, want := range tests {
		if got := epicColor(json.RawMessage(value)); got != want {
			t.Errorf("epicColor(%s) = %q, want %q", value, got, want)
		}
	}
}
