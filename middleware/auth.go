package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gbrlsnchs/jwt/v3"
	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/database"
	"github.com/staticbackendhq/core/model"
)

const (
	RootRole = 100
)

func RequireAuth(datastore database.Persister, volatile cache.Volatilizer) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("Authorization")

			if len(key) == 0 {
				// if they requested a public repo we let them continue
				// to next security check.
				if strings.HasPrefix(r.URL.Path, "/db/pub_") || strings.HasPrefix(r.URL.Path, "/query/pub_") {
					a := model.Auth{
						AccountID: "public_repo_called",
						UserID:    "public_repo_called",
						Email:     "",
						Role:      0,
						Token:     "pub",
						Plan:      model.PlanIdea,
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
				err = fmt.Errorf("error validating auth key: %w", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			ctx = context.WithValue(ctx, ContextAuth, auth)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ValidateAuthKey(datastore database.Persister, volatile cache.Volatilizer, ctx context.Context, key string) (model.Auth, error) {
	a := model.Auth{}

	var pl model.JWTPayload
	if _, err := jwt.Verify([]byte(key), model.HashSecret, &pl); err != nil {
		return a, fmt.Errorf("could not verify your authentication token: %s", err.Error())
	}

	conf, ok := ctx.Value(ContextBase).(model.DatabaseConfig)
	if !ok {
		return a, fmt.Errorf("invalid StaticBackend public token")
	}

	var auth model.Auth
	if err := volatile.GetTyped(pl.Token, &auth); err == nil {
		return auth, nil
	}

	parts := strings.Split(key, "|")
	if len(parts) != 2 {
		return a, fmt.Errorf("invalid authentication token")
	}

	token, err := datastore.FindUser(conf.Name, parts[0], parts[1])
	if err != nil {
		return a, fmt.Errorf("error retrieving your token: %s", err.Error())
	}

	// TODO: This was datastore.FindAccount(token.AccountID) before the
	// backend refactor, this is very strange and should not have worked.....
	// I changed it to use the tenant's ID from current database, which was what
	// would have made sense.  can't explain why the previous datastore.FindAccount
	// could find the "customer" via the user's Account ID, does not make ANY sense
	cus, err := datastore.FindTenant(conf.TenantID)
	if err != nil {
		return a, fmt.Errorf("error retrieving your customer account: %v", err)
	}

	a = model.Auth{
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

func RequireRoot(datastore database.Persister, volatile cache.Volatilizer) Middleware {
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

			// in dev mode the cache will have a key called:
			// dev-root-token which hold the dynamically changing root token
			// which changes each stop/start of the CLI. Using
			// "safe-to-use-in-dev-root-token" as root token will
			// get the real root token removing the need to update the root token
			// in dev mode
			if key == "safe-to-use-in-dev-root-token" {
				rt, err := volatile.Get("dev-root-token")
				if err != nil {
					http.Error(w, "not in dev mode", http.StatusUnauthorized)
					return
				}

				key = rt
			}

			ctx := r.Context()
			conf, ok := ctx.Value(ContextBase).(model.DatabaseConfig)
			if !ok {
				http.Error(w, "invalid StaticBackend public key", http.StatusBadRequest)
				return
			}

			tok, err := ValidateRootToken(datastore, conf.Name, key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			a := model.Auth{
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

func ValidateRootToken(datastore database.Persister, base, token string) (model.User, error) {
	tok := model.User{}

	parts := strings.Split(token, "|")
	if len(parts) != 3 {
		return tok, fmt.Errorf("invalid root token")
	}

	id := parts[0]
	acctID := parts[1]
	token = parts[2]

	tok, err := datastore.FindRootUser(base, id, acctID, token)
	if err != nil {
		return tok, err
	} else if tok.Role < RootRole {
		return tok, fmt.Errorf("not enough permission")
	}
	return tok, nil
}
