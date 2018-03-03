// Package middleware implements http handler middlewares.
package middleware

import (
	"io"
	"net/http"

	"github.com/gorilla/handlers"
)

// Middleware wraps a Handler with some pre- and/or post actions.
type Middleware func(http.Handler) http.Handler

// SetResponseHeader sets the specified response header to a fixed value.
func SetResponseHeader(header, value string) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(header, value)
			h.ServeHTTP(w, r)
		})
	}
}

// RequestLogger logs requests to out in Apache Common Log Format (CLF).
func RequestLogger(out io.Writer) Middleware {
	return func(h http.Handler) http.Handler {
		return handlers.LoggingHandler(out, h)
	}
}
