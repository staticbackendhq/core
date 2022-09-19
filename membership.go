package staticbackend

import (
	"encoding/json"
	"errors"
	"fmt"

	"math/rand"
	"net/http"
	"strings"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/email"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/middleware"
	"github.com/staticbackendhq/core/model"

	"golang.org/x/crypto/bcrypt"
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

	l.Email = strings.ToLower(l.Email)

	tok, err := m.validateUserPassword(conf.Name, l.Email, l.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jwtBytes, err := m.getAuthToken(tok, conf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, string(jwtBytes))
}

func (m *membership) validateUserPassword(dbName, email, password string) (tok model.User, err error) {
	email = strings.ToLower(email)

	tok, err = backend.DB.FindUserByEmail(dbName, email)
	if err != nil {
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(tok.Password), []byte(password)); err != nil {
		return tok, errors.New("invalid email/password")
	}

	return
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

	l.Email = strings.ToLower(l.Email)

	exists, err := backend.DB.UserEmailExists(conf.Name, l.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if exists {
		http.Error(w, "invalid email", http.StatusBadRequest)
		return
	}

	jwtBytes, tok, err := m.createAccountAndUser(conf.Name, l.Email, l.Password, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token := string(jwtBytes)

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
	if err := backend.Cache.SetTyped("base:"+token, conf); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, token)
}

func (m *membership) getAuthToken(tok model.User, conf model.DatabaseConfig) (jwtBytes []byte, err error) {
	token := fmt.Sprintf("%s|%s", tok.ID, tok.Token)

	// get their JWT
	jwtBytes, err = backend.GetJWT(token)
	if err != nil {
		return
	}

	auth := model.Auth{
		AccountID: tok.AccountID,
		UserID:    tok.ID,
		Email:     tok.Email,
		Role:      tok.Role,
		Token:     tok.Token,
	}

	//TODO: find a good way to find all occurences of those two
	// and make them easily callable via a shared function
	if err = backend.Cache.SetTyped(token, auth); err != nil {
		return
	}
	if err = backend.Cache.SetTyped("base:"+token, conf); err != nil {
		return
	}

	return
}

func (m *membership) createAccountAndUser(dbName, email, password string, role int) ([]byte, model.User, error) {
	acctID, err := backend.DB.CreateAccount(dbName, email)
	if err != nil {
		return nil, model.User{}, err
	}

	jwtBytes, tok, err := m.createUser(dbName, acctID, email, password, role)
	if err != nil {
		return nil, model.User{}, err
	}
	return jwtBytes, tok, nil
}

func (m *membership) createUser(dbName, accountID, email, password string, role int) ([]byte, model.User, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, model.User{}, err
	}

	tok := model.User{
		AccountID: accountID,
		Email:     email,
		Token:     backend.DB.NewID(),
		Password:  string(b),
		Role:      role,
	}

	tokID, err := backend.DB.CreateUser(dbName, tok)
	if err != nil {
		return nil, model.User{}, err
	}

	tok.ID = tokID

	token := fmt.Sprintf("%s|%s", tokID, tok.Token)

	// Get their JWT
	jwtBytes, err := backend.GetJWT(token)
	if err != nil {
		return nil, tok, err
	}

	auth := model.Auth{
		AccountID: tok.AccountID,
		UserID:    tok.ID,
		Email:     tok.Email,
		Role:      role,
		Token:     tok.Token,
	}
	if err := backend.Cache.SetTyped(token, auth); err != nil {
		return nil, tok, err
	}

	return jwtBytes, tok, nil
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

	tok, err := backend.DB.FindUserByEmail(conf.Name, email)
	if err != nil {
		http.Error(w, "email not found", http.StatusNotFound)
		return
	}

	if err := backend.DB.SetPasswordResetCode(conf.Name, tok.ID, code); err != nil {
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

	data.Email = strings.ToLower(data.Email)

	b, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := backend.DB.ResetPassword(conf.Name, data.Email, data.Code, string(b)); err != nil {
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

	data.Email = strings.ToLower(data.Email)

	if err := backend.DB.SetUserRole(conf.Name, data.Email, data.Role); err != nil {
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

	tok, err := m.validateUserPassword(conf.Name, data.Email, data.OldPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	newpw, err := bcrypt.GenerateFromPassword([]byte(data.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := backend.DB.UserSetPassword(conf.Name, tok.ID, string(newpw)); err != nil {
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

	if r.Method == http.MethodGet {
		// we use GET to validate magic link code
		email := r.URL.Query().Get("email")
		code := r.URL.Query().Get("code")

		val, err := backend.Cache.Get("ml-" + email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		parts := strings.Split(val, " ")
		if len(parts) != 2 {
			http.Error(w, "invalid data", http.StatusBadRequest)
			return
		}

		// if the code isn't what was set we make sure they're not trying to
		// "brute force" random code.
		if parts[0] != code {
			if len(parts[1]) >= 10 {
				http.Error(w, "maximum retry reched", http.StatusTooManyRequests)
				return
			}

			if err := backend.Cache.Set("ml-"+email, val+"a"); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			respond(w, http.StatusBadRequest, false)
			return
		}

		// they got the right code, return a session token

		tok, err := backend.DB.FindUserByEmail(conf.Name, email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		jwtBytes, err := m.getAuthToken(tok, conf)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		respond(w, http.StatusOK, string(jwtBytes))
		return
	}

	data := new(struct {
		FromEmail string `json:"fromEmail"`
		FromName  string `json:"fromName"`
		Email     string `json:"email"`
		Subject   string `json:"subject"`
		Body      string `json:"body"`
		MagicLink string `json:"link"`
	})
	if err := parseBody(r.Body, &data); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	code := rand.Intn(987654) + 123456
	// to accomodate unit test, we hard code a magic link code in dev mode
	if config.Current.AppEnv == AppEnvDev {
		code = 666333
	}
	data.MagicLink += fmt.Sprintf("?code=%d&email=%s", code, data.Email)

	if err := backend.Cache.Set("ml-"+data.Email, fmt.Sprintf("%d a", code)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mail := email.SendMailData{
		From:     data.FromEmail,
		FromName: data.FromName,
		To:       data.Email,
		Subject:  data.Subject,
		HTMLBody: strings.Replace(data.Body, "[link]", data.MagicLink, -1),
	}
	if err := backend.Emailer.Send(mail); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}
