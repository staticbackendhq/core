package middleware

import (
	"net/http"
)

// Middleware is a standard http.Handler
type Middleware func(h http.Handler) http.Handler

// Chain creates a request pipeline from which the Middleware are chained
// together and h is the last Handler executed.
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
