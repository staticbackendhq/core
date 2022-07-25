package staticbackend

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/middleware"

	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/facebook"
	"github.com/markbates/goth/providers/google"
	"github.com/markbates/goth/providers/twitter"
)

const (
	OAuthProviderTwitter  = "twitter"
	OAuthProviderFacebook = "facebook"
	OAuthProviderGoogle   = "google"
)

type ExternalLogins struct {
	membership *membership
}

func (el *ExternalLogins) login() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provider := r.URL.Query().Get("provider")
		reqID := r.URL.Query().Get("reqid")

		if len(reqID) <= 5 {
			http.Error(w, "reqid parameters is required to be > 5", http.StatusBadRequest)
			return
		}

		conf, _, err := middleware.Extract(r, false)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := volatile.SetTyped("oauth_"+reqID, conf); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		customer, err := datastore.FindAccount(conf.CustomerID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		info, ok := customer.GetProvider(provider)
		if !ok {
			e := fmt.Sprintf("missing configuration for provider: %s", provider)
			http.Error(w, e, http.StatusNotFound)
			return
		}

		p, err := el.getProvider(conf.ID, provider, reqID, info)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, err := p.BeginAuth(reqID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			url, err := sess.GetAuthURL()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := volatile.SetTyped(reqID+"_session", sess.Marshal()); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, url, http.StatusTemporaryRedirect)
		})

		next.ServeHTTP(w, r)
	})
}

func (el *ExternalLogins) callback() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provider := r.URL.Query().Get("provider")
		reqID := r.URL.Query().Get("reqid")

		var conf internal.BaseConfig
		if err := volatile.GetTyped("oauth_"+reqID, &conf); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		customer, err := datastore.FindAccount(conf.CustomerID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		info, ok := customer.GetProvider(provider)
		if !ok {
			e := fmt.Sprintf("missing configuration for provider: %s", provider)
			http.Error(w, e, http.StatusNotFound)
			return
		}

		p, err := el.getProvider(conf.ID, provider, reqID, info)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var value string
		if err := volatile.GetTyped(reqID+"_session", &value); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sess, err := p.UnmarshalSession(value)
		if err := volatile.GetTyped(reqID+"_session", &value); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			params := r.URL.Query()
			if params.Encode() == "" && r.Method == "POST" {
				r.ParseForm()
				params = r.Form
			}

			// get new token and retry fetch
			_, err = sess.Authorize(p, params)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			//TODO: should we store this in cache?

			user, err := p.FetchUser(sess)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			sessionToken, err := el.registerOrLogin(conf.Name, provider, user.Email, user.AccessToken)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := volatile.Set("token_"+reqID, sessionToken); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// TODO: Could be nice to pass as much properties from the goth.User (user variable)
			// via either an event or a cache value that the caller receives or
			// request to grab for instance user's name, avatar etc.

			render(w, r, "oauth.html", nil, nil)
		})

		next.ServeHTTP(w, r)
	})
}

func (el *ExternalLogins) twitter(dbName, provider, reqID string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionToken, err := el.registerOrLogin(dbName, provider, "todo", "todo")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := volatile.Set("token_"+reqID, sessionToken); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		render(w, r, "oauth.html", nil, nil)
	})
}

func (el *ExternalLogins) registerOrLogin(dbName, provider, email, accessToken string) (sessionToken string, err error) {
	email = strings.ToLower(email)

	exists, err := datastore.UserEmailExists(dbName, email)
	if err != nil {
		return
	}

	if exists {
		return el.signIn(dbName, email)
	}

	return el.signUp(dbName, provider, email, accessToken)
}

func (el *ExternalLogins) signIn(dbName, email string) (sessionToken string, err error) {
	tok, err := datastore.FindTokenByEmail(dbName, email)
	if err != nil {
		return
	}

	token := fmt.Sprintf("%s|%s", tok.ID, tok.Token)

	b, err := el.membership.getJWT(token)
	if err != nil {
		return
	}

	sessionToken = string(b)
	return
}

func (el *ExternalLogins) signUp(dbName, provider, email, accessToken string) (sessionToken string, err error) {
	pw := fmt.Sprintf("%s:%s", provider, accessToken)

	b, _, err := el.membership.createAccountAndUser(dbName, email, pw, 100)
	if err != nil {
		return
	}

	sessionToken = string(b)
	return
}

func (el *ExternalLogins) getProvider(dbID, provider, reqID string, info internal.OAuthConfig) (p goth.Provider, err error) {
	callbackURL := fmt.Sprintf(
		"http://localhost:8099/oauth/callback?provider=%s&reqid=%s&sbpk=%s",
		provider,
		reqID,
		dbID,
	)

	if provider == OAuthProviderTwitter {
		return twitter.New(info.ConsumerKey, info.ConsumerSecret, callbackURL), nil
	} else if provider == OAuthProviderFacebook {
		return facebook.New(info.ConsumerKey, info.ConsumerSecret, callbackURL), nil
	} else if provider == OAuthProviderGoogle {
		return google.New(info.ConsumerKey, info.ConsumerSecret, callbackURL), nil
	}
	return twitter.New("", "", ""), errors.New("invalid auth provider")
}

func (*ExternalLogins) getState(r *http.Request) string {
	params := r.URL.Query()
	if params.Encode() == "" && r.Method == http.MethodPost {
		return r.FormValue("state")
	}
	return params.Get("state")
}
