package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/avanrish/dashboard-for-jira/internal/config"
	"github.com/avanrish/dashboard-for-jira/internal/jira"
	"github.com/avanrish/dashboard-for-jira/internal/web"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	settings, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(settings.Port)))
	if err != nil {
		return err
	}

	boards := jira.BoardSource{
		Client:  jira.NewClient(&url.URL{Scheme: "https", Host: settings.JiraSite}, settings.JiraEmail, settings.JiraAPIToken),
		BoardID: settings.JiraBoardID,
	}
	httpServer := &http.Server{
		Handler:           web.NewHandler(boards, logger),
		ReadHeaderTimeout: 5 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- httpServer.Serve(listener) }()
	logger.Info("serving the dashboard", "url", "http://"+listener.Addr().String())

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
