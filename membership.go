package staticbackend

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/middleware"

	"golang.org/x/crypto/bcrypt"

	"github.com/gbrlsnchs/jwt/v3"
)

type membership struct {
	volatile internal.Volatilizer
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

	exists, err := datastore.UserEmailExists(conf.Name, email)
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

	var l internal.Login
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

	token := fmt.Sprintf("%s|%s", tok.ID, tok.Token)

	// get their JWT
	jwtBytes, err := m.getJWT(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	auth := internal.Auth{
		AccountID: tok.AccountID,
		UserID:    tok.ID,
		Email:     tok.Email,
		Role:      tok.Role,
		Token:     tok.Token,
	}

	//TODO: find a good way to find all occurences of those two
	// and make them easily callable via a shared function
	if err := m.volatile.SetTyped(token, auth); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := m.volatile.SetTyped("base:"+token, conf); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, string(jwtBytes))
}

func (m *membership) validateUserPassword(dbName, email, password string) (tok internal.Token, err error) {
	email = strings.ToLower(email)

	tok, err = datastore.FindTokenByEmail(dbName, email)
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
		log.Println("invalid StaticBackend key")
		return
	}

	var l internal.Login
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	l.Email = strings.ToLower(l.Email)

	exists, err := datastore.UserEmailExists(conf.Name, l.Email)
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

	auth := internal.Auth{
		AccountID: tok.AccountID,
		UserID:    tok.ID,
		Email:     tok.Email,
		Role:      tok.Role,
		Token:     tok.Token,
	}

	if err := m.volatile.SetTyped(token, auth); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := m.volatile.SetTyped("base:"+token, conf); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, token)
}

func (m *membership) createAccountAndUser(dbName, email, password string, role int) ([]byte, internal.Token, error) {
	acctID, err := datastore.CreateUserAccount(dbName, email)
	if err != nil {
		return nil, internal.Token{}, err
	}

	jwtBytes, tok, err := m.createUser(dbName, acctID, email, password, role)
	if err != nil {
		return nil, internal.Token{}, err
	}
	return jwtBytes, tok, nil
}

func (m *membership) createUser(dbName, accountID, email, password string, role int) ([]byte, internal.Token, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, internal.Token{}, err
	}

	tok := internal.Token{
		AccountID: accountID,
		Email:     email,
		Token:     datastore.NewID(),
		Password:  string(b),
		Role:      role,
	}

	tokID, err := datastore.CreateUserToken(dbName, tok)
	if err != nil {
		return nil, internal.Token{}, err
	}

	tok.ID = tokID

	token := fmt.Sprintf("%s|%s", tokID, tok.Token)

	// Get their JWT
	jwtBytes, err := m.getJWT(token)
	if err != nil {
		return nil, tok, err
	}

	auth := internal.Auth{
		AccountID: tok.AccountID,
		UserID:    tok.ID,
		Email:     tok.Email,
		Role:      role,
		Token:     tok.Token,
	}
	if err := m.volatile.SetTyped(token, auth); err != nil {
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

	code := randStringRunes(10)

	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	tok, err := datastore.FindTokenByEmail(conf.Name, email)
	if err != nil {
		http.Error(w, "email not found", http.StatusNotFound)
		return
	}

	if err := datastore.SetPasswordResetCode(conf.Name, tok.ID, code); err != nil {
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

	if err := datastore.ResetPassword(conf.Name, data.Email, data.Code, string(b)); err != nil {
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

	if err := datastore.SetUserRole(conf.Name, data.Email, data.Role); err != nil {
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

	if err := datastore.UserSetPassword(conf.Name, tok.ID, string(newpw)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

func (m *membership) getJWT(token string) ([]byte, error) {
	now := time.Now()
	pl := internal.JWTPayload{
		Payload: jwt.Payload{
			Issuer:         "StaticBackend",
			ExpirationTime: jwt.NumericDate(now.Add(12 * time.Hour)),
			NotBefore:      jwt.NumericDate(now.Add(30 * time.Minute)),
			IssuedAt:       jwt.NumericDate(now),
			JWTID:          randStringRunes(32), // changed from primitive.NewObjectID
		},
		Token: token,
	}

	return jwt.Sign(pl, internal.HashSecret)

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

	tok, err := datastore.GetFirstTokenFromAccountID(conf.Name, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	token := fmt.Sprintf("%s|%s", tok.ID, tok.Token)

	jwtBytes, err := m.getJWT(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	auth := internal.Auth{
		AccountID: tok.AccountID,
		UserID:    tok.ID,
		Email:     tok.Email,
		Role:      tok.Role,
		Token:     tok.Token,
	}
	if err := m.volatile.SetTyped(token, auth); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, string(jwtBytes))
}
