package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/staticbackendhq/core/internal"

	"github.com/gbrlsnchs/jwt/v3"
)

const (
	RootRole = 100
)

func RequireAuth(datastore internal.Persister, volatile internal.PubSuber) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("Authorization")

			if len(key) == 0 {
				// if they requested a public repo we let them continue
				// to next security check.
				if strings.HasPrefix(r.URL.Path, "/db/pub_") || strings.HasPrefix(r.URL.Path, "/query/pub_") {
					a := internal.Auth{
						AccountID: "public_repo_called",
						UserID:    "public_repo_called",
						Email:     "",
						Role:      0,
						Token:     "pub",
						Plan:      internal.PlanIdea,
					}

					ctx := context.WithValue(r.Context(), ContextAuth, a)

					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}

				http.Error(w, "missing authorization HTTP header", http.StatusUnauthorized)
				return
			} else if !strings.HasPrefix(key, "Bearer ") {
				http.Error(w,
					fmt.Sprintf("invalid authorization HTTP header, should be: Bearer your-token, but we got %s", key),
					http.StatusBadRequest,
				)
				return
			}

			key = strings.Replace(key, "Bearer ", "", -1)

			ctx := r.Context()

			auth, err := ValidateAuthKey(datastore, volatile, ctx, key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			ctx = context.WithValue(ctx, ContextAuth, auth)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ValidateAuthKey(datastore internal.Persister, volatile internal.PubSuber, ctx context.Context, key string) (internal.Auth, error) {
	a := internal.Auth{}

	var pl internal.JWTPayload
	if _, err := jwt.Verify([]byte(key), internal.HashSecret, &pl); err != nil {
		return a, fmt.Errorf("could not verify your authentication token: %s", err.Error())
	}

	conf, ok := ctx.Value(ContextBase).(internal.BaseConfig)
	if !ok {
		return a, fmt.Errorf("invalid StaticBackend public token")
	}

	var auth internal.Auth
	if err := volatile.GetTyped(pl.Token, &auth); err == nil {
		return auth, nil
	}

	parts := strings.Split(key, "|")
	if len(parts) != 2 {
		return a, fmt.Errorf("invalid authentication token")
	}

	token, err := datastore.FindToken(conf.Name, parts[0], parts[1])
	if err != nil {
		return a, fmt.Errorf("error retrieving your token: %s", err.Error())
	}

	cus, err := datastore.FindAccount(token.AccountID)
	if err != nil {
		return a, fmt.Errorf("error retrieving your customer account: %v", err)
	}

	a = internal.Auth{
		AccountID: token.AccountID,
		UserID:    token.ID,
		Email:     token.Email,
		Role:      token.Role,
		Token:     token.Token,
		Plan:      cus.Plan,
	}
	if err := volatile.SetTyped(pl.Token, a); err != nil {
		return a, err
	}

	// set base:token useful when executing pubsub event message / function
	if err := volatile.SetTyped("base:"+pl.Token, conf); err != nil {
		return a, err
	}

	return a, nil
}

func RequireRoot(datastore internal.Persister) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("Authorization")

			// we check if the token is in a cookie (used from UI)
			if len(key) == 0 {
				ck, err := r.Cookie("token")
				if err == nil || ck != nil {
					key = fmt.Sprintf("Bearer %s", ck.Value)
				}
			}

			if len(key) == 0 {
				http.Error(w, "missing authorization HTTP header", http.StatusUnauthorized)
				return
			} else if strings.HasPrefix(key, "Bearer ") == false {
				http.Error(w,
					fmt.Sprintf("invalid authorization HTTP header, should be: Bearer your-token, but we got %s", key),
					http.StatusBadRequest,
				)
				return
			}

			key = strings.Replace(key, "Bearer ", "", -1)

			ctx := r.Context()
			conf, ok := ctx.Value(ContextBase).(internal.BaseConfig)
			if !ok {
				http.Error(w, "invalid StaticBackend public key", http.StatusBadRequest)
				return
			}

			tok, err := ValidateRootToken(datastore, conf.Name, key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			a := internal.Auth{
				AccountID: tok.AccountID,
				UserID:    tok.ID,
				Email:     tok.Email,
				Role:      tok.Role,
				Token:     tok.Token,
			}

			ctx = context.WithValue(ctx, ContextAuth, a)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ValidateRootToken(datastore internal.Persister, base, token string) (internal.Token, error) {
	tok := internal.Token{}

	parts := strings.Split(token, "|")
	if len(parts) != 3 {
		return tok, fmt.Errorf("invalid root token")
	}

	id := parts[0]
	acctID := parts[1]
	token = parts[2]

	tok, err := datastore.FindRootToken(base, id, acctID, token)
	if err != nil {
		return tok, err
	} else if tok.Role < RootRole {
		return tok, fmt.Errorf("not enough permission")
	}
	return tok, nil
}
