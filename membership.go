package staticbackend

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"net/http"
	"strings"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/middleware"
	"github.com/staticbackendhq/core/model"
)

type membership struct {
	//volatile internal.Volatilizer
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

	token, err := mship.Authenticate(l.Email, l.Password, l.AccountID)
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
		slog.Error("invalid StaticBackend key", "error", err)
		return
	}

	var l model.Login
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mship := backend.Membership(conf)
	token, err := mship.Register(l.Email, l.Password, l.AccountID)
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
	if err != nil || a.Role < 50 {
		http.Error(w, "insufficient priviledges", http.StatusUnauthorized)
		return
	}

	var data = new(struct {
		AccountID string `json:"accountId"`
		Email     string `json:"email"`
		Role      int    `json:"role"`
	})
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(data.Email) == 0 {
		http.Error(w, "missing account id or email", http.StatusBadRequest)
		return
	} else if data.Role > 50 {
		http.Error(w, "role cannot be > than 50", http.StatusBadRequest)
		return
	}

	if len(data.AccountID) == 0 {
		// if no account id specify we default to current authenticated user
		data.AccountID = a.AccountID
	}

	mship := backend.Membership(conf)
	if err := mship.SetUserRole(data.AccountID, data.Email, data.Role); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

/*
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
*/

func (m *membership) sudoGetTokenFromAccountID(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := getURLPart(r.URL.Path, 2)

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

func (m *membership) getAuthTokenByUserID(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	accountID := getURLPart(r.URL.Path, 2)
	userID := getURLPart(r.URL.Path, 3)
	if len(accountID) == 0 || len(userID) == 0 {
		http.Error(w, "missing account id or user id", http.StatusBadRequest)
		return
	}

	mship := backend.Membership(conf)
	user, err := mship.GetUserByID(accountID, userID)
	if err != nil || user.AccountID != accountID {
		assoc, assocErr := backend.DB.GetAccountUser(conf.Name, userID, accountID)
		if assocErr != nil {
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			http.Error(w, "user not found for account", http.StatusNotFound)
			return
		}

		user = model.User{
			ID:        assoc.UserID,
			AccountID: assoc.AccountID,
			Email:     assoc.Email,
			Role:      assoc.Role,
			Token:     assoc.Token,
			Created:   assoc.Created,
		}
	}

	jwtBytes, err := mship.GetAuthToken(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	respond(w, http.StatusOK, string(jwtBytes))
}

func (m *membership) getUserByID(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	accountID := getURLPart(r.URL.Path, 2)
	userID := getURLPart(r.URL.Path, 3)
	if len(accountID) == 0 || len(userID) == 0 {
		http.Error(w, "missing account id or user id", http.StatusBadRequest)
		return
	}

	mship := backend.Membership(conf)
	user, err := mship.GetUserByID(accountID, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	respond(w, http.StatusOK, user)
}

func (m *membership) me(w http.ResponseWriter, r *http.Request) {
	_, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	respond(w, http.StatusOK, auth)
}

func (m *membership) changeEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var data = new(struct {
		Email string `json:"email"`
	})
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data.Email = strings.ToLower(strings.TrimSpace(data.Email))
	if len(data.Email) == 0 || !strings.Contains(data.Email, "@") || !strings.Contains(data.Email, ".") {
		http.Error(w, "invalid email", http.StatusBadRequest)
		return
	}

	mship := backend.Membership(conf)
	if err := mship.ChangeEmail(auth, data.Email); err != nil {
		if errors.Is(err, backend.ErrEmailAlreadyInUse) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

func (m *membership) deleteAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	conf, auth, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if auth.Role < 50 {
		http.Error(w, "insufficient priviledges", http.StatusUnauthorized)
		return
	}

	files, err := backend.DB.ListAllFiles(conf.Name, auth.AccountID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, file := range files {
		if err := backend.Filestore.Delete(file.Key); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := backend.DB.DeleteFile(conf.Name, file.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := backend.DB.DeleteAccount(conf.Name, auth.AccountID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
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
