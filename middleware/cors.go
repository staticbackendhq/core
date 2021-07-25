package middleware

import (
	"net/http"
	"strings"
)

// Cors enables calls via remote origin to handle external JavaScript calls mainly.
func Cors() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headers := w.Header()
			origin := r.Header.Get("Origin")

			// Always set Vary headers
			// see https://github.com/rs/cors/issues/10,
			//     https://github.com/rs/cors/commit/dbdca4d95feaa7511a46e6f1efb3b3aa505bc43f#commitcomment-12352001
			headers.Add("Vary", "Origin")
			headers.Add("Vary", "Access-Control-Request-Method")
			headers.Add("Vary", "Access-Control-Request-Headers")

			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			headers.Set("Access-Control-Allow-Origin", origin)
			// Spec says: Since the list of methods can be unbounded, simply returning the method indicated
			// by Access-Control-Request-Method (if supported) can be enough
			headers.Set("Access-Control-Allow-Methods", strings.ToUpper(r.Header.Get("Access-Control-Request-Method")))

			headers.Set("Access-Control-Allow-Headers", r.Header.Get("Access-Control-Request-Headers"))

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
