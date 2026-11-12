package web

import (
	"context"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/avanrish/dashboard-for-jira/internal/board"
)

type emptySource struct{}

func (emptySource) Board(context.Context) (board.Board, error) {
	return board.Board{}, nil
}

type fixedSource board.Board

func (s fixedSource) Board(context.Context) (board.Board, error) {
	return board.Board(s), nil
}

func newTestHandler() http.Handler {
	return NewHandler(emptySource{}, slog.New(slog.DiscardHandler))
}

func TestOnlyLocalHostsAllowed(t *testing.T) {
	tests := []struct {
		host string
		want int
	}{
		{"localhost:8080", http.StatusOK},
		{"127.0.0.1:8080", http.StatusOK},
		{"[::1]:8080", http.StatusOK},
		{"localhost", http.StatusOK},
		{"evil.example", http.StatusForbidden},
		{"evil.example:8080", http.StatusForbidden},
		{"localhost.evil.example:8080", http.StatusForbidden},
	}
	handler := newTestHandler()
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			request.Host = tt.host
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != tt.want {
				t.Errorf("status = %d, want %d", recorder.Code, tt.want)
			}
		})
	}
}

func TestCrossOriginWritesRejected(t *testing.T) {
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "http://localhost:8080/", nil)
	request.Header.Set("Sec-Fetch-Site", "cross-site")
	recorder := httptest.NewRecorder()
	newTestHandler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestCollapsedEpicsRenderClosed(t *testing.T) {
	source := fixedSource{
		Columns: []string{"To Do"},
		Lanes: []board.Lane{
			{Epic: board.Epic{Key: "ABC-1"}, Cells: [][]board.Issue{nil}},
			{Epic: board.Epic{Key: "ABC-2"}, Cells: [][]board.Issue{nil}},
			{Cells: [][]board.Issue{nil}},
		},
	}
	tests := []struct {
		name   string
		cookie string
		want   map[string]bool
	}{
		{"no cookie", "", map[string]bool{"ABC-1": true, "ABC-2": true, "no-epic": true}},
		{"none collapsed", "collapsed_epics=", map[string]bool{"ABC-1": true, "ABC-2": true, "no-epic": true}},
		{"some collapsed", "collapsed_epics=ABC-2,no-epic", map[string]bool{"ABC-1": true, "ABC-2": false, "no-epic": false}},
		{"unknown epic", "collapsed_epics=XYZ-9", map[string]bool{"ABC-1": true, "ABC-2": true, "no-epic": true}},
	}
	handler := NewHandler(source, slog.New(slog.DiscardHandler))
	detailsTag := regexp.MustCompile(`<details[^>]*>`)
	epicAttribute := regexp.MustCompile(`data-epic="([^"]*)"`)
	openAttribute := regexp.MustCompile(`\sopen[\s>]`)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "http://localhost:8080/", nil)
			if tt.cookie != "" {
				request.Header.Set("Cookie", tt.cookie)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
			}

			open := map[string]bool{}
			for _, tag := range detailsTag.FindAllString(recorder.Body.String(), -1) {
				epic := epicAttribute.FindStringSubmatch(tag)
				if epic == nil {
					t.Fatalf("epic without data-epic: %s", tag)
				}
				open[epic[1]] = openAttribute.MatchString(tag)
			}
			if !maps.Equal(open, tt.want) {
				t.Errorf("open epics = %v, want %v", open, tt.want)
			}
		})
	}
}
