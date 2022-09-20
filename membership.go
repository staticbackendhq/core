package staticbackend

import (
	"encoding/json"
	"fmt"

	"net/http"
	"strings"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/middleware"
	"github.com/staticbackendhq/core/model"
)

type membership struct {
	//volatile internal.Volatilizer
	log *logger.Logger
}

func (m *membership) emailExists(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(r.URL.Query().Get("e"))
	if len(email) == 0 {
		respond(w, http.StatusOK, false)
		return
	}

	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, "invalid StaticBackend key", http.StatusUnauthorized)
		return
	}

	exists, err := backend.DB.UserEmailExists(conf.Name, email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, exists)
}

func (m *membership) login(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, "invalid StaticBackend key", http.StatusUnauthorized)
		return
	}

	var l model.Login
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mship := backend.Membership(conf)

	token, err := mship.Authenticate(l.Email, l.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	respond(w, http.StatusOK, token)
}

func (m *membership) register(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, "invalid StaticBackend key", http.StatusUnauthorized)
		m.log.Error().Err(err).Msg("invalid StaticBackend key")
		return
	}

	var l model.Login
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mship := backend.Membership(conf)
	token, err := mship.Register(l.Email, l.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, token)
}

func (m *membership) setResetCode(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(r.URL.Query().Get("e"))
	if len(email) == 0 || strings.Index(email, "@") <= 0 {
		http.Error(w, "invalid email", http.StatusBadRequest)
		return
	}

	code := internal.RandStringRunes(10)

	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	mship := backend.Membership(conf)
	if err := mship.SetPasswordResetCode(email, code); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, code)
}

func (m *membership) resetPassword(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var data = new(struct {
		Email    string `json:"email"`
		Code     string `json:"code"`
		Password string `json:"password"`
	})
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mship := backend.Membership(conf)
	if err := mship.ResetPassword(data.Email, data.Code, data.Password); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

func (m *membership) setRole(w http.ResponseWriter, r *http.Request) {
	conf, a, err := middleware.Extract(r, true)
	if err != nil || a.Role < 100 {
		http.Error(w, "insufficient priviledges", http.StatusUnauthorized)
		return
	}

	var data = new(struct {
		Email string `json:"email"`
		Role  int    `json:"role"`
	})
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mship := backend.Membership(conf)
	if err := mship.SetUserRole(data.Email, data.Role); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

func (m *membership) setPassword(w http.ResponseWriter, r *http.Request) {
	conf, a, err := middleware.Extract(r, true)
	if err != nil || a.Role < 100 {
		http.Error(w, "insufficient priviledges", http.StatusUnauthorized)
		return
	}

	var data = new(struct {
		Email       string `json:"email"`
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	})
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mship := backend.Membership(conf)
	if err := mship.UserSetPassword(data.Email, data.OldPassword, data.NewPassword); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

func (m *membership) sudoGetTokenFromAccountID(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := ""

	_, r.URL.Path = ShiftPath(r.URL.Path)
	id, r.URL.Path = ShiftPath(r.URL.Path)

	tok, err := backend.DB.GetFirstUserFromAccountID(conf.Name, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token := fmt.Sprintf("%s|%s", tok.ID, tok.Token)

	jwtBytes, err := backend.GetJWT(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	auth := model.Auth{
		AccountID: tok.AccountID,
		UserID:    tok.ID,
		Email:     tok.Email,
		Role:      tok.Role,
		Token:     tok.Token,
	}
	if err := backend.Cache.SetTyped(token, auth); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, string(jwtBytes))
}

func (m *membership) me(w http.ResponseWriter, r *http.Request) {
	_, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	respond(w, http.StatusOK, auth)
}

func (m *membership) magicLink(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, "invalid StaticBackend key", http.StatusUnauthorized)
		return
	}

	mship := backend.Membership(conf)

	if r.Method == http.MethodGet {
		// we use GET to validate magic link code
		email := r.URL.Query().Get("email")
		code := r.URL.Query().Get("code")

		token, err := mship.ValidateMagicLink(email, code)
		if err != nil {
			if strings.Contains(err.Error(), "maximum") {
				http.Error(w, err.Error(), http.StatusTooManyRequests)
				return
			}

			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		respond(w, http.StatusOK, token)
		return
	}

	var data backend.MagicLinkData
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := mship.SetupMagicLink(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}
