package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"github.com/avanrish/dashboard-for-jira/internal/config"
	"github.com/avanrish/dashboard-for-jira/internal/configure"
	"github.com/avanrish/dashboard-for-jira/internal/jira"
	"github.com/avanrish/dashboard-for-jira/internal/web"
)

const usage = `A web dashboard for Jira, served on 127.0.0.1.

Usage:
  dashboard-for-jira <command>

Commands:
  start      serve the dashboard, first asking for any missing settings
  configure  change the Jira site, email address, API token, board, or port
  version    print the version
  help       show this list
`

func main() {
	flag.Usage = func() { _, _ = fmt.Fprint(flag.CommandLine.Output(), usage) }
	flag.Parse()

	var err error
	switch command := flag.Arg(0); {
	case flag.NArg() > 1:
		flag.Usage()
		os.Exit(2)
	case command == "version":
		fmt.Println(version())
	case command == "" || command == "help":
		fmt.Print(usage)
	case command == "start":
		err = start()
	case command == "configure":
		// No signal handling here: Ctrl-C ends the program while it waits for an answer.
		err = configure.Run(context.Background())
	default:
		_, _ = fmt.Fprintf(os.Stderr, "unknown command %q\n\n", command)
		flag.Usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// version returns the version Go recorded in the binary: the release tag, such as v0.1.0, for go install,
// or a pseudo-version naming the commit for a build from a checkout.
func version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Main.Version
	}
	return "unknown"
}

// start serves the dashboard, first asking in the terminal for any setting that is missing.
func start() error {
	settings, err := config.Load()
	var settingsErr *config.SettingsError
	if errors.As(err, &settingsErr) && len(settingsErr.Missing) > 0 && configure.CanAsk() {
		settings, err = configure.Missing(context.Background(), settingsErr.Missing)
	}
	if err != nil {
		return err
	}
	return serve(settings)
}

func serve(settings config.Config) error {
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
	logger.Info("serving the dashboard", "url", (&url.URL{Scheme: "http", Host: listener.Addr().String()}).String())

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
