package web

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avanrish/dashboard-for-jira/internal/board"
)

type emptySource struct{}

func (emptySource) Board(context.Context) (board.Board, error) {
	return board.Board{}, nil
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
