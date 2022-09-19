package backend

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gbrlsnchs/jwt/v3"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/model"
	"golang.org/x/crypto/bcrypt"
)

// User handles everything related to accounts and users inside a database
type User struct {
	conf model.DatabaseConfig
}

func newUser(baseID string) User {
	return User{conf: findBase(baseID)}
}

// CreateAccount creates a new account in this database
func (u User) CreateAccount(email string) (string, error) {
	email = strings.ToLower(email)
	if exists, err := DB.UserEmailExists(u.conf.Name, email); err != nil {
		return "", err
	} else if exists {
		return "", errors.New("email not available")
	}

	return DB.CreateAccount(u.conf.Name, email)
}

// CreateUserToken creates a user token (login) for a specific account in a database
func (u User) CreateUserToken(accountID, email, password string, role int) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	tok := model.User{
		AccountID: accountID,
		Email:     email,
		Password:  string(b),
		Token:     DB.NewID(),
		Role:      role,
		Created:   time.Now(),
	}
	return DB.CreateUser(u.conf.Name, tok)
}

// Authenticate tries to authenticate an email/password and return a session token
func (u User) Authenticate(email, password string) (string, error) {
	tok, err := DB.FindUserByEmail(u.conf.Name, email)
	if err != nil {
		return "", err
	}

	if err = bcrypt.CompareHashAndPassword([]byte(tok.Password), []byte(password)); err != nil {
		return "", errors.New("invalid email/password")
	}

	token := fmt.Sprintf("%s|%s", tok.ID, tok.Token)

	jwt, err := GetJWT(token)
	if err != nil {
		return "", err
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
	if err = Cache.SetTyped(token, auth); err != nil {
		return "", err
	}
	if err = Cache.SetTyped("base:"+token, u.conf); err != nil {
		return "", err
	}

	return string(jwt), nil
}

// SetPasswordResetCode sets the password forget code for a user
func (u User) SetPasswordResetCode(tokenID, code string) error {
	return DB.SetPasswordResetCode(u.conf.Name, tokenID, code)
}

// ResetPassword resets the password of a matching email/code for a user
func (u User) ResetPassword(email, code, password string) error {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return DB.ResetPassword(u.conf.Name, email, code, string(b))
}

// SetUserRole changes the role of a user
func (u User) SetUserRole(email string, role int) error {
	return DB.SetUserRole(u.conf.Name, email, role)
}

// UserSetPassword password changes initiated by the user
func (u User) UserSetPassword(tokenID, password string) error {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return DB.UserSetPassword(u.conf.Name, tokenID, string(b))
}

func GetJWT(token string) ([]byte, error) {
	now := time.Now()
	pl := model.JWTPayload{
		Payload: jwt.Payload{
			Issuer:         "StaticBackend",
			ExpirationTime: jwt.NumericDate(now.Add(12 * time.Hour)),
			NotBefore:      jwt.NumericDate(now.Add(30 * time.Minute)),
			IssuedAt:       jwt.NumericDate(now),
			JWTID:          internal.RandStringRunes(32),
		},
		Token: token,
	}

	return jwt.Sign(pl, model.HashSecret)

}
