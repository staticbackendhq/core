package backend

import (
	"errors"
	"fmt"
	"strings"

	sb "github.com/staticbackendhq/core"
	"github.com/staticbackendhq/core/model"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	conf model.BaseConfig
}

func newUser(baseID string) User {
	return User{conf: findBase(baseID)}
}

func (u User) CreateAccount(email string) (string, error) {
	email = strings.ToLower(email)
	if exists, err := datastore.UserEmailExists(u.conf.Name, email); err != nil {
		return "", err
	} else if exists {
		return "", errors.New("email not available")
	}

	return datastore.CreateUserAccount(u.conf.Name, email)
}

func (u User) CreateUserToken(tok model.Token) (string, error) {
	return datastore.CreateUserToken(u.conf.Name, tok)
}

func (u User) Authenticate(email, password string) (string, error) {
	tok, err := datastore.FindTokenByEmail(u.conf.Name, email)
	if err != nil {
		return "", err
	}

	if err = bcrypt.CompareHashAndPassword([]byte(tok.Password), []byte(password)); err != nil {
		return "", errors.New("invalid email/password")
	}

	token := fmt.Sprintf("%s|%s", tok.ID, tok.Token)

	jwt, err := sb.GetJWT(token)
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
