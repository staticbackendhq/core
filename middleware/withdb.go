package middleware

import (
	"context"
	"net/http"

	"github.com/staticbackendhq/core/internal"
)

func WithDB(datastore internal.Persister, volatile internal.PubSuber) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("SB-PUBLIC-KEY")

			// we check in query string (used for SSE)
			if len(key) == 0 {
				key = r.URL.Query().Get("sbpk")
			}

			// we check in cookie (used via the UI)
			if len(key) == 0 {
				ck, err := r.Cookie("pk")
				if err == nil || ck != nil {
					key = ck.Value
				}
			}

			if len(key) == 0 {
				http.Error(w, "invalid StaticBackend public key", http.StatusUnauthorized)
				return
			}

			ctx := r.Context()

			var conf internal.BaseConfig
			if err := volatile.GetTyped(key, &conf); err == nil {
				ctx = context.WithValue(ctx, ContextBase, conf)
			} else {
				// let's try to see if they are allow to use a database
				conf, err = datastore.FindDatabase(key)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				} else if !conf.IsActive {
					http.Error(w, "your account is not inactive. Please contact us support@staticbackend.com", http.StatusUnauthorized)
					return
				}

				if err := volatile.SetTyped(key, conf); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

				ctx = context.WithValue(ctx, ContextBase, conf)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
