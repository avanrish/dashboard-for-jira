package web

import (
	"net/http"
	"net/url"
)

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
