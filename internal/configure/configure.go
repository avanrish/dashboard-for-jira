// Package configure asks for the dashboard's settings in the terminal, checks them against Jira, and saves them.
package configure

import (
	"bufio"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/avanrish/dashboard-for-jira/internal/config"
	"github.com/avanrish/dashboard-for-jira/internal/jira"
)

// Run asks for every setting, showing the current values, then checks the settings against Jira and saves them.
func Run(ctx context.Context) error {
	s, err := newSession()
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(s.out, "Settings go to %s. Press Enter to keep a value in brackets.\n\n", s.path)
	if _, err := s.run(ctx, func(string) bool { return true }); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(s.out, "Run %s start to open the dashboard.\n", config.AppName)
	return nil
}

// Missing asks for the settings named in missing, by their environment variables such as JIRA_SITE,
// then checks the settings against Jira, saves them, and returns them.
func Missing(ctx context.Context, missing []string) (config.Config, error) {
	s, err := newSession()
	if err != nil {
		return config.Config{}, err
	}
	_, _ = fmt.Fprintf(s.out, "Some settings are missing; your answers go to %s.\n\n", s.path)
	return s.run(ctx, func(name string) bool { return slices.Contains(missing, name) })
}

// CanAsk reports whether standard input is a terminal, where someone can answer questions.
func CanAsk() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

type session struct {
	path       string // the settings file
	in         *bufio.Reader
	out        io.Writer
	readSecret func() (string, error) // reads a line without showing it
	siteURL    func(site string) *url.URL
}

func newSession() (session, error) {
	path, err := config.Path()
	if err != nil {
		return session{}, err
	}
	in := bufio.NewReader(os.Stdin)
	return session{
		path:       path,
		in:         in,
		out:        os.Stdout,
		readSecret: func() (string, error) { return readSecret(in, os.Stdout) },
		siteURL:    func(site string) *url.URL { return &url.URL{Scheme: "https", Host: site} },
	}, nil
}

// run asks for each setting that asking accepts, by its environment variable's name,
// then checks the settings against Jira and saves them.
func (s session) run(ctx context.Context, asking func(name string) bool) (config.Config, error) {
	file, err := config.ReadFile(s.path)
	if err != nil {
		_, _ = fmt.Fprintf(s.out, "The current settings cannot be read, so they will be replaced: %v\n\n", err)
		file = config.File{}
	}
	previousEmail := file.Email

	if asking("JIRA_SITE") {
		if file.Site, err = ask(s, "JIRA_SITE", "Jira site, such as example.atlassian.net", file.Site, parseSite); err != nil {
			return config.Config{}, err
		}
	}
	if asking("JIRA_EMAIL") {
		if file.Email, err = ask(s, "JIRA_EMAIL", "Email address of your Atlassian account", file.Email, parseEmail); err != nil {
			return config.Config{}, err
		}
	}
	var token string // stays empty to keep the saved token
	if asking("JIRA_API_TOKEN") {
		if os.Getenv("JIRA_API_TOKEN") != "" {
			_, _ = fmt.Fprintln(s.out, "API token: using JIRA_API_TOKEN, which takes precedence over a saved token")
		} else if token, err = s.askToken(cmp.Or(os.Getenv("JIRA_EMAIL"), file.Email)); err != nil {
			return config.Config{}, err
		}
	}
	if asking("JIRA_BOARD_ID") {
		if file.BoardID, err = ask(s, "JIRA_BOARD_ID", "Board ID, or the board's address", formatNumber(file.BoardID), parseBoardID); err != nil {
			return config.Config{}, err
		}
	}
	if asking("PORT") {
		if file.Port, err = ask(s, "PORT", "Port to serve the dashboard on", formatNumber(cmp.Or(file.Port, 8080)), parsePort); err != nil {
			return config.Config{}, err
		}
	}

	settings, err := config.Resolve(file, token)
	if err != nil {
		return config.Config{}, err
	}

	_, _ = fmt.Fprintln(s.out, "\nChecking with Jira...")
	client := jira.NewClient(s.siteURL(settings.JiraSite), settings.JiraEmail, settings.JiraAPIToken)

	name, err := client.Myself(ctx)
	if err != nil {
		return config.Config{}, fmt.Errorf("signing in to %s failed, so nothing was saved: %w", settings.JiraSite, err)
	}

	board, err := client.BoardConfiguration(ctx, settings.JiraBoardID)
	if err != nil {
		return config.Config{}, fmt.Errorf("opening board %d failed, so nothing was saved: %w", settings.JiraBoardID, err)
	}
	_, _ = fmt.Fprintf(s.out, "Signed in as %s. Found board %q.\n\n", name, board.Name)

	if err := file.Save(s.path); err != nil {
		return config.Config{}, err
	}
	_, _ = fmt.Fprintf(s.out, "Saved the settings to %s.\n", s.path)

	if token != "" {
		if err := config.SaveToken(settings.JiraEmail, token); err != nil {
			return config.Config{}, fmt.Errorf("saving the API token to the system's password store failed: %w\nSet JIRA_API_TOKEN in the environment instead", err)
		}
		_, _ = fmt.Fprintln(s.out, "Saved the API token to the system's password store.")

		if previousEmail != "" && previousEmail != settings.JiraEmail {
			if err := config.DeleteToken(previousEmail); err != nil {
				return config.Config{}, fmt.Errorf("removing the API token saved for %s: %w", previousEmail, err)
			}
		}
	}
	return settings, nil
}

// ask prints question with the current value in brackets, then reads answers until parse accepts one.
// An empty answer keeps the current value. name is the setting's environment variable.
func ask[T any](s session, name, question, current string, parse func(answer string) (T, error)) (T, error) {
	if os.Getenv(name) != "" {
		_, _ = fmt.Fprintf(s.out, "%s is set in the environment, which takes precedence over this answer.\n", name)
	}
	for {
		if current == "" {
			_, _ = fmt.Fprintf(s.out, "%s: ", question)
		} else {
			_, _ = fmt.Fprintf(s.out, "%s [%s]: ", question, current)
		}
		answer, err := readLine(s.in)
		if err != nil {
			var zero T
			return zero, err
		}
		if answer == "" {
			answer = current
		}
		value, err := parse(answer)
		if err == nil {
			return value, nil
		}
		_, _ = fmt.Fprintf(s.out, "%v.\n", err)
	}
}

// askToken reads the API token without showing it.
// An empty answer keeps the token saved for email, if there is one, and returns "".
func (s session) askToken(email string) (string, error) {
	saved, err := config.SavedToken(email)
	hasSaved := err == nil && saved != ""
	question := "API token, from https://id.atlassian.com/manage-profile/security/api-tokens"
	if hasSaved {
		question += " [saved]"
	}
	for {
		_, _ = fmt.Fprintf(s.out, "%s: ", question)
		token, err := s.readSecret()
		if err != nil {
			return "", err
		}
		if token != "" || hasSaved {
			return token, nil
		}
		_, _ = fmt.Fprintln(s.out, "Enter an API token.")
	}
}

// formatNumber formats a number setting for a question, where 0 means it is not set.
func formatNumber(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// readLine reads one line, without surrounding spaces.
func readLine(in *bufio.Reader) (string, error) {
	line, err := in.ReadString('\n')
	if errors.Is(err, io.EOF) {
		if line == "" {
			return "", errors.New("input ended before every question was answered")
		}
		err = nil
	}
	return strings.TrimSpace(line), err
}

// readSecret reads a line from the terminal without showing it, or from in when standard input is not a terminal.
func readSecret(in *bufio.Reader, out io.Writer) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return readLine(in)
	}
	state, err := term.GetState(fd)
	if err != nil {
		return "", err
	}
	// term.ReadPassword turns echo off until it returns; turn it back on if Ctrl-C ends the program first.
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	done := make(chan struct{})
	defer func() {
		signal.Stop(interrupts)
		close(done)
	}()
	go func() {
		select {
		case <-interrupts:
			_ = term.Restore(fd, state)
			_, _ = fmt.Fprintln(out)
			os.Exit(130) // the status shells report for a program ended by Ctrl-C
		case <-done:
		}
	}()
	secret, err := term.ReadPassword(fd)
	_, _ = fmt.Fprintln(out) // the Enter key was not shown either
	return strings.TrimSpace(string(secret)), err
}

// parseSite accepts a site's host name, or any address on the site, such as the board's.
func parseSite(answer string) (string, error) {
	host, _, _ := strings.Cut(answer, "/")
	if strings.Contains(answer, "://") {
		if address, err := url.Parse(answer); err == nil {
			host = address.Host
		}
	}
	if host == "" || strings.Contains(host, ":") {
		return "", errors.New("enter the site's host name, such as example.atlassian.net")
	}
	return host, nil
}

func parseEmail(answer string) (string, error) {
	if !strings.Contains(answer, "@") {
		return "", errors.New("enter the email address you sign in to Atlassian with")
	}
	return answer, nil
}

// parseBoardID accepts a board's ID, or its address, such as https://example.atlassian.net/jira/software/projects/ABC/boards/12.
func parseBoardID(answer string) (int, error) {
	if address, err := url.Parse(answer); err == nil && address.Host != "" {
		segments := strings.Split(address.Path, "/")
		if i := slices.Index(segments, "boards"); i >= 0 && i+1 < len(segments) {
			answer = segments[i+1]
		}
	}
	if id, err := strconv.Atoi(answer); err == nil && id > 0 {
		return id, nil
	}
	return 0, errors.New("enter the board's ID, the number after /boards/ in its address, or paste the address")
}

func parsePort(answer string) (int, error) {
	if port, err := strconv.Atoi(answer); err == nil && port >= 1 && port <= 65535 {
		return port, nil
	}
	return 0, errors.New("enter a port number from 1 to 65535")
}
