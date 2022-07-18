package staticbackend

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/middleware"

	"golang.org/x/oauth2"
)

const (
	OAuthProviderTwitter = "twitter"
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

		oauthConf, err := el.getOauthConfByProvider(provider, reqID, info)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uri := oauthConf.AuthCodeURL(reqID)
			fmt.Println("DEBUG: redirecting to twitter: ", uri)
			http.Redirect(w, r, uri, http.StatusTemporaryRedirect)
		})

		next.ServeHTTP(w, r)
	})
}

func (el *ExternalLogins) callback() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provider := getURLPart(r.URL.Path, 3)
		reqID := getURLPart(r.URL.Path, 4)

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

		oauthConf, err := el.getOauthConfByProvider(provider, reqID, info)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")

			token, err := oauthConf.Exchange(context.Background(), code)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			fmt.Println(token)
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

func (el *ExternalLogins) getOauthConfByProvider(provider, reqID string, info internal.OAuthConfig) (c *oauth2.Config, err error) {
	callbackURL := fmt.Sprintf("http://127.0.0.1/oauth/%s/%s", provider, reqID)

	if provider == OAuthProviderTwitter {
		c = &oauth2.Config{
			ClientID:     info.ConsumerKey,
			ClientSecret: info.ConsumerSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://api.twitter.com/oauth/request_token",
				TokenURL: "https://api.twitter.com/oauth/access_token",
			},
			RedirectURL: callbackURL,
		}
	} else {
		err = fmt.Errorf("cannot find keys for provider %s", provider)
	}
	return
}
