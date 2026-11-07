package web

import (
	"bytes"
	"context"
	"embed"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/a-h/templ"

	"github.com/avanrish/dashboard-for-jira/internal/board"
)

//go:embed static
var staticFiles embed.FS

type BoardSource interface {
	Board(ctx context.Context) (board.Board, error)
}

type server struct {
	boards BoardSource
	logger *slog.Logger
}

func NewHandler(boards BoardSource, logger *slog.Logger) http.Handler {
	s := &server{boards: boards, logger: logger}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.FileServerFS(staticFiles))
	mux.HandleFunc("GET /{$}", s.handlePage)

	return allowLocalHost(http.NewCrossOriginProtection().Handler(mux))
}

func (s *server) handlePage(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	boardData, err := s.boards.Board(r.Context())
	duration := time.Since(start).Round(time.Millisecond)
	switch {
	case r.Context().Err() != nil:
		s.logger.InfoContext(r.Context(), "the browser left before the board loaded", "duration", duration)
	case err != nil:
		s.logger.ErrorContext(r.Context(), "loading the board", "duration", duration, "err", err)
		// Not 502: templ's dev proxy retries those for minutes, taking them for a restarting server.
		s.render(w, r, http.StatusInternalServerError, errorPage(err.Error()))
	default:
		s.logger.InfoContext(r.Context(), "loaded the board", "duration", duration)
		s.render(w, r, http.StatusOK, page(boardData))
	}
}

// render buffers the output so a template error becomes a clean 500, not a truncated page.
func (s *server) render(w http.ResponseWriter, r *http.Request, status int, component templ.Component) {
	var buf bytes.Buffer
	if err := component.Render(r.Context(), &buf); err != nil {
		s.logger.ErrorContext(r.Context(), "rendering a page", "err", err)
		http.Error(w, "Could not render the page.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

// allowLocalHost blocks DNS rebinding, where another domain is made to resolve to 127.0.0.1.
func allowLocalHost(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch (&url.URL{Host: r.Host}).Hostname() {
		case "localhost", "127.0.0.1", "::1":
			next.ServeHTTP(w, r)
		default:
			http.Error(w, "This server only answers requests for localhost.", http.StatusForbidden)
		}
	})
}
