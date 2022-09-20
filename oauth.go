package staticbackend

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/middleware"
	"github.com/staticbackendhq/core/model"

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
	log *logger.Logger
}

type ExternalUser struct {
	Token     string `json:"token"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	FirstName string `json:"first"`
	LastName  string `json:"last"`
	AvatarURL string `json:"avatarUrl"`
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

		if err := backend.Cache.SetTyped("oauth_"+reqID, conf); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		customer, err := backend.DB.FindTenant(conf.TenantID)
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
			sess, err := p.BeginAuth(el.toState(provider, reqID, conf.ID))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			url, err := sess.GetAuthURL()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			if err := backend.Cache.SetTyped(reqID+"_session", sess.Marshal()); err != nil {
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
		provider, reqID, baseID := el.fromState(el.getState(r))

		var conf model.DatabaseConfig
		if err := backend.Cache.GetTyped("oauth_"+reqID, &conf); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else if conf.ID != baseID {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		customer, err := backend.DB.FindTenant(conf.TenantID)
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
		if err := backend.Cache.GetTyped(reqID+"_session", &value); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sess, err := p.UnmarshalSession(value)
		if err := backend.Cache.GetTyped(reqID+"_session", &value); err != nil {
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

			user, err := p.FetchUser(sess)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			accessTokens := fmt.Sprintf("%s|%s", user.AccessToken, user.AccessTokenSecret)
			sessionToken, err := el.registerOrLogin(conf, provider, user.Email, accessTokens)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			extuser := ExternalUser{
				Token:     sessionToken,
				Email:     user.Email,
				Name:      user.Name,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				AvatarURL: user.AvatarURL,
			}

			if err := backend.Cache.SetTyped("extuser_"+reqID, extuser); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			render(w, r, "oauth.html", nil, nil, el.log)
		})

		next.ServeHTTP(w, r)
	})
}

func (*ExternalLogins) getUser(w http.ResponseWriter, r *http.Request) {
	reqID := r.URL.Query().Get("reqid")

	var extuser ExternalUser
	if err := backend.Cache.GetTyped("extuser_"+reqID, &extuser); err != nil {
		respond(w, http.StatusOK, false)
		return
	}

	respond(w, http.StatusOK, extuser)
}

func (el *ExternalLogins) registerOrLogin(conf model.DatabaseConfig, provider, email, accessToken string) (sessionToken string, err error) {
	email = strings.ToLower(email)

	exists, err := backend.DB.UserEmailExists(conf.Name, email)
	if err != nil {
		return
	}

	if exists {
		return el.signIn(conf, email)
	}

	return el.signUp(conf, provider, email, accessToken)
}

func (el *ExternalLogins) signIn(conf model.DatabaseConfig, email string) (sessionToken string, err error) {
	tok, err := backend.DB.FindUserByEmail(conf.Name, email)
	if err != nil {
		return
	}

	token := fmt.Sprintf("%s|%s", tok.ID, tok.Token)

	b, err := backend.GetJWT(token)
	if err != nil {
		return
	}

	sessionToken = string(b)
	return
}

func (el *ExternalLogins) signUp(conf model.DatabaseConfig, provider, email, accessToken string) (sessionToken string, err error) {
	pw := fmt.Sprintf("%s:%s", provider, accessToken)

	mship := backend.Membership(conf)

	b, _, err := mship.CreateAccountAndUser(email, pw, 0)
	if err != nil {
		return
	}

	sessionToken = string(b)
	return
}

func (el *ExternalLogins) getProvider(dbID, provider, reqID string, info model.OAuthConfig) (p goth.Provider, err error) {
	callbackURL := fmt.Sprintf(
		"%s/oauth/callback",
		config.Current.AppURL,
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

func (*ExternalLogins) toState(provider, reqID, baseID string) string {
	return fmt.Sprintf("%s_%s_%s", provider, reqID, baseID)
}

func (*ExternalLogins) fromState(state string) (provider, reqID, baseID string) {
	parts := strings.Split(state, "_")
	if len(parts) != 3 {
		return
	}

	provider = parts[0]
	reqID = parts[1]
	baseID = parts[2]
	return
}
