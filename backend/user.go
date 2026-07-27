package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/gbrlsnchs/jwt/v3"
	"github.com/staticbackendhq/core/email"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/model"
	"golang.org/x/crypto/bcrypt"
)

const systemAccountTrigger = "sys-sb_accounts"

var ErrEmailAlreadyInUse = errors.New("email already in use")

// User handles everything related to accounts and users inside a database
type User struct {
	conf model.DatabaseConfig
}

func newUser(base model.DatabaseConfig) User {
	return User{conf: base}
}

// Authenticate tries to authenticate an email/password and return a session token.
// An optional accountID can be provided to log into a cross-account association
// instead of the user's home account.
func (u User) Authenticate(email, password string, accountID ...string) (string, error) {
	email = strings.ToLower(email)

	tok, err := DB.FindUserByEmail(u.conf.Name, email)
	if err != nil {
		return "", err
	}

	if err = bcrypt.CompareHashAndPassword([]byte(tok.Password), []byte(password)); err != nil {
		return "", errors.New("invalid email/password")
	}

	var auth model.Auth

	// if an accountID is provided and differs from the home account, look up the association
	if len(accountID) > 0 && accountID[0] != "" && accountID[0] != tok.AccountID {
		exists, err := DB.AssociationExists(u.conf.Name, tok.ID, accountID[0])
		if err != nil {
			return "", err
		}
		if !exists {
			assoc := model.AccountUser{
				UserID:    tok.ID,
				AccountID: accountID[0],
				Email:     tok.Email,
				Role:      0,
				Token:     DB.NewID(),
			}
			if _, err := DB.AddAccountUser(u.conf.Name, assoc); err != nil {
				return "", err
			}
		}

		assoc, err := DB.GetAccountUser(u.conf.Name, tok.ID, accountID[0])
		if err != nil {
			return "", errors.New("invalid email/password")
		}

		token := fmt.Sprintf("%s|%s", tok.ID, assoc.Token)

		jwtBytes, err := GetJWT(token)
		if err != nil {
			return "", err
		}

		auth = model.Auth{
			AccountID: assoc.AccountID,
			UserID:    tok.ID,
			Email:     assoc.Email,
			Role:      assoc.Role,
			Token:     assoc.Token,
		}

		if err = Cache.SetTyped(token, auth); err != nil {
			return "", err
		}
		if err = Cache.SetTyped("base:"+token, u.conf); err != nil {
			return "", err
		}

		return string(jwtBytes), nil
	}

	token := fmt.Sprintf("%s|%s", tok.ID, tok.Token)

	jwtBytes, err := GetJWT(token)
	if err != nil {
		return "", err
	}

	auth = model.Auth{
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

	return string(jwtBytes), nil
}

// Register creates a new account and user.
// An optional accountID can be provided when the email already exists in the schema
// but the user wants to join an additional account. In that case the password is
// verified against the existing record and a cross-account association is created.
func (u User) Register(email, password string, accountID ...string) (string, error) {
	email = strings.ToLower(email)

	exists, err := DB.UserEmailExists(u.conf.Name, email)
	if err != nil {
		return "", err
	}

	if exists {
		// only allowed when an explicit accountID is supplied
		if len(accountID) == 0 || accountID[0] == "" {
			return "", errors.New("invalid email")
		}

		// verify credentials against the existing record
		tok, err := DB.FindUserByEmail(u.conf.Name, email)
		if err != nil {
			return "", err
		}
		if err = bcrypt.CompareHashAndPassword([]byte(tok.Password), []byte(password)); err != nil {
			return "", errors.New("invalid email/password")
		}

		// refuse if already associated with this account
		if tok.AccountID == accountID[0] {
			return "", errors.New("already a member of this account")
		}
		if _, err := DB.GetAccountUser(u.conf.Name, tok.ID, accountID[0]); err == nil {
			return "", errors.New("already a member of this account")
		}

		assoc := model.AccountUser{
			UserID:    tok.ID,
			AccountID: accountID[0],
			Email:     tok.Email,
			Role:      0,
			Token:     DB.NewID(),
		}
		if _, err := DB.AddAccountUser(u.conf.Name, assoc); err != nil {
			return "", err
		}

		token := fmt.Sprintf("%s|%s", tok.ID, assoc.Token)
		jwtBytes, err := GetJWT(token)
		if err != nil {
			return "", err
		}

		auth := model.Auth{
			AccountID: assoc.AccountID,
			UserID:    tok.ID,
			Email:     assoc.Email,
			Role:      assoc.Role,
			Token:     assoc.Token,
		}
		if err := Cache.SetTyped(token, auth); err != nil {
			return "", err
		}
		if err := Cache.SetTyped("base:"+token, u.conf); err != nil {
			return "", err
		}

		return string(jwtBytes), nil
	}

	// account creator has role=50 (Account Admin)
	jwtBytes, tok, err := u.CreateAccountAndUser(email, password, 50)
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
	u.publishAccountCreated(acctID, email, tok)
	return jwtBytes, tok, nil
}

func (u User) publishAccountCreated(accountID, email string, tok model.User) {
	auth := model.Auth{
		AccountID: tok.AccountID,
		UserID:    tok.ID,
		Email:     tok.Email,
		Role:      tok.Role,
		Token:     tok.Token,
	}

	data := map[string]interface{}{
		"id":        accountID,
		"email":     email,
		"userId":    tok.ID,
		"userEmail": tok.Email,
		"userRole":  tok.Role,
	}
	b, err := json.Marshal(data)
	if err != nil {
		slog.Error("error marshaling system account event", "error", err)
		return
	}

	if err := Cache.Publish(model.Command{
		Channel: systemAccountTrigger,
		Data:    string(b),
		Type:    model.MsgTypeDBCreated,
		Auth:    auth,
		Base:    u.conf.Name,
	}); err != nil {
		slog.Error("error publishing system account event", "error", err)
	}
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

// SetUserRole changes the role of a user's membership in a specific account.
func (u User) SetUserRole(accountID, email string, role int) error {
	email = strings.ToLower(email)
	return DB.SetUserRole(u.conf.Name, accountID, email, role)
}

// ChangeEmail changes the authenticated user's email address.
func (u User) ChangeEmail(auth model.Auth, newEmail string) error {
	newEmail = strings.ToLower(newEmail)
	oldEmail := strings.ToLower(auth.Email)

	if newEmail == oldEmail {
		return nil
	}

	exists, err := DB.UserEmailExists(u.conf.Name, newEmail)
	if err != nil {
		return err
	}
	if exists {
		return ErrEmailAlreadyInUse
	}

	if err := DB.ChangeUserEmail(u.conf.Name, auth.UserID, auth.AccountID, oldEmail, newEmail); err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("%s|%s", auth.UserID, auth.Token)
	if err := Cache.Delete(cacheKey); err != nil {
		return err
	}

	tok, err := DB.FindUserByEmail(u.conf.Name, newEmail)
	if err != nil {
		return err
	}
	homeCacheKey := fmt.Sprintf("%s|%s", tok.ID, tok.Token)
	if homeCacheKey != cacheKey {
		return Cache.Delete(homeCacheKey)
	}
	return nil
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

// GetUserByID returns a user by account and user IDs.
func (u User) GetUserByID(accountID, userID string) (model.User, error) {
	return DB.GetUserByID(u.conf.Name, accountID, userID)
}

// PromoteToOwnAccount moves a user from being a member of someone else's account
// to having their own home account, while preserving the old membership as an
// association in sb_account_users.
func (u User) PromoteToOwnAccount(auth model.Auth) (string, error) {
	// create the user's own account
	newAcctID, err := DB.CreateAccount(u.conf.Name, auth.Email)
	if err != nil {
		return "", err
	}

	// preserve the old membership as an association
	assoc := model.AccountUser{
		UserID:    auth.UserID,
		AccountID: auth.AccountID,
		Email:     auth.Email,
		Role:      auth.Role,
		Token:     DB.NewID(),
	}
	if _, err := DB.AddAccountUser(u.conf.Name, assoc); err != nil {
		return "", err
	}

	// move the user's home account and grant Admin role (50)
	if err := DB.UpdateUserAccount(u.conf.Name, auth.UserID, newAcctID, 50); err != nil {
		return "", err
	}

	// the old cached token will expire at its natural 12h TTL;
	// the caller should treat the returned JWT as the new session token

	// build and cache a fresh token for the new home account
	updatedUser := model.User{
		ID:        auth.UserID,
		AccountID: newAcctID,
		Email:     auth.Email,
		Role:      50,
		Token:     auth.Token,
	}
	jwtBytes, err := u.GetAuthToken(updatedUser)
	if err != nil {
		return "", err
	}

	return string(jwtBytes), nil
}

// GetJWT returns a session token from a token
func GetJWT(token string) ([]byte, error) {
	now := time.Now()
	pl := model.JWTPayload{
		Payload: jwt.Payload{
			Issuer:         "StaticBackend",
			ExpirationTime: jwt.NumericDate(now.Add(12 * time.Hour)),
			NotBefore:      jwt.NumericDate(now),
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
		HTMLBody: strings.ReplaceAll(data.Body, "[link]", data.MagicLink),
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
