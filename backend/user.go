package backend

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gbrlsnchs/jwt/v3"
	"github.com/staticbackendhq/core/email"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/model"
	"golang.org/x/crypto/bcrypt"
)

// User handles everything related to accounts and users inside a database
type User struct {
	conf model.DatabaseConfig
}

func newUser(base model.DatabaseConfig) User {
	return User{conf: base}
}

// Authenticate tries to authenticate an email/password and return a session token
func (u User) Authenticate(email, password string) (string, error) {
	email = strings.ToLower(email)

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

// Register creates a new account and user
func (u User) Register(email, password string) (string, error) {
	email = strings.ToLower(email)

	exists, err := DB.UserEmailExists(u.conf.Name, email)
	if err != nil {
		return "", err
	} else if exists {
		return "", errors.New("invalid email")
	}

	jwtBytes, tok, err := u.CreateAccountAndUser(email, password, 0)
	if err != nil {
		return "", err
	}

	token := string(jwtBytes)

	auth := model.Auth{
		AccountID: tok.AccountID,
		UserID:    tok.ID,
		Email:     tok.Email,
		Role:      tok.Role,
		Token:     tok.Token,
	}

	if err := Cache.SetTyped(token, auth); err != nil {
		return "", err
	}
	if err := Cache.SetTyped("base:"+token, u.conf); err != nil {
		return "", err
	}

	return token, nil
}

// CreateAccountAndUser creates an account with a user
func (u User) CreateAccountAndUser(email, password string, role int) ([]byte, model.User, error) {
	acctID, err := DB.CreateAccount(u.conf.Name, email)
	if err != nil {
		return nil, model.User{}, err
	}

	jwtBytes, tok, err := u.CreateUser(acctID, email, password, role)
	if err != nil {
		return nil, model.User{}, err
	}
	return jwtBytes, tok, nil
}

// CreateUser creates a user for an Account
func (u User) CreateUser(accountID, email, password string, role int) ([]byte, model.User, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, model.User{}, err
	}

	tok := model.User{
		AccountID: accountID,
		Email:     email,
		Token:     DB.NewID(),
		Password:  string(b),
		Role:      role,
	}

	tokID, err := DB.CreateUser(u.conf.Name, tok)
	if err != nil {
		return nil, model.User{}, err
	}

	tok.ID = tokID

	token := fmt.Sprintf("%s|%s", tokID, tok.Token)

	// Get their JWT
	jwtBytes, err := GetJWT(token)
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
	if err := Cache.SetTyped(token, auth); err != nil {
		return nil, tok, err
	}

	return jwtBytes, tok, nil
}

// SetPasswordResetCode sets the password forget code for a user
func (u User) SetPasswordResetCode(email, code string) error {
	email = strings.ToLower(email)

	tok, err := DB.FindUserByEmail(u.conf.Name, email)
	if err != nil {
		return err
	}

	if err := DB.SetPasswordResetCode(u.conf.Name, tok.ID, code); err != nil {
		return err
	}
	return nil
}

// ResetPassword resets the password of a matching email/code for a user
func (u User) ResetPassword(email, code, password string) error {
	email = strings.ToLower(email)

	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return DB.ResetPassword(u.conf.Name, email, code, string(b))
}

// SetUserRole changes the role of a user
func (u User) SetUserRole(email string, role int) error {
	email = strings.ToLower(email)
	return DB.SetUserRole(u.conf.Name, email, role)
}

// UserSetPassword password changes initiated by the user
func (u User) UserSetPassword(email, oldpw, newpw string) error {
	email = strings.ToLower(email)

	tok, err := DB.FindUserByEmail(u.conf.Name, email)
	if err != nil {
		return err
	}

	if _, err := u.Authenticate(email, oldpw); err != nil {
		return err
	}

	b, err := bcrypt.GenerateFromPassword([]byte(newpw), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return DB.UserSetPassword(u.conf.Name, tok.ID, string(b))
}

// GetAuthToken returns a session token for a user
func (u User) GetAuthToken(tok model.User) (jwtBytes []byte, err error) {
	token := fmt.Sprintf("%s|%s", tok.ID, tok.Token)

	// get their JWT
	jwtBytes, err = GetJWT(token)
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
	if err = Cache.SetTyped(token, auth); err != nil {
		return
	}
	if err = Cache.SetTyped("base:"+token, u.conf); err != nil {
		return
	}

	return
}

// GetJWT returns a session token from a token
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

// MagicLinkData magic links for no-password sign-in
type MagicLinkData struct {
	FromEmail string `json:"fromEmail"`
	FromName  string `json:"fromName"`
	Email     string `json:"email"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	MagicLink string `json:"link"`
}

// SetupMagicLink initialize a magic link and send the email to the user
func (u User) SetupMagicLink(data MagicLinkData) error {
	data.Email = strings.ToLower(data.Email)

	code := rand.Intn(987654) + 123456
	//TODO: the constant AppEnv should be moved to the config package?
	// to accomodate unit test, we hard code a magic link code in dev mode
	if Config.AppEnv == "dev" {
		code = 666333
	}
	data.MagicLink += fmt.Sprintf("?code=%d&email=%s", code, data.Email)

	if err := Cache.Set("ml-"+data.Email, fmt.Sprintf("%d a", code)); err != nil {
		return err
	}

	mail := email.SendMailData{
		From:     data.FromEmail,
		FromName: data.FromName,
		To:       data.Email,
		Subject:  data.Subject,
		HTMLBody: strings.Replace(data.Body, "[link]", data.MagicLink, -1),
	}
	if err := Emailer.Send(mail); err != nil {
		return err
	}
	return nil
}

// ValidateMagicLink validates a magic link code and returns a session token on
// success
func (u User) ValidateMagicLink(email, code string) (string, error) {
	email = strings.ToLower(email)

	val, err := Cache.Get("ml-" + email)
	if err != nil {
		return "", err
	}

	parts := strings.Split(val, " ")
	if len(parts) != 2 {
		return "", errors.New("invalid code")
	}

	// if the code isn't what was set we make sure they're not trying to
	// "brute force" random code.
	if parts[0] != code {
		if len(parts[1]) >= 10 {
			return "", errors.New("maximum retry reached")
		}

		if err := Cache.Set("ml-"+email, val+"a"); err != nil {
			return "", err
		}
	}

	// they got the right code, return a session token

	tok, err := DB.FindUserByEmail(u.conf.Name, email)
	if err != nil {
		return "", err
	}

	jwtBytes, err := u.GetAuthToken(tok)
	if err != nil {
		return "", err
	}

	return string(jwtBytes), nil
}
