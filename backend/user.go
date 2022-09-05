package backend

import (
	"errors"
	"strings"

	"github.com/staticbackendhq/core/internal"
)

type User struct{}

func (u User) CreateAccount(dbName, email string) (string, error) {
	email = strings.ToLower(email)
	if exists, err := datastore.UserEmailExists(dbName, email); err != nil {
		return "", err
	} else if exists {
		return "", errors.New("email not available")
	}

	return datastore.CreateUserAccount(dbName, email)
}

func (u User) CreateUserToken(dbName string, tok internal.Token) (string, error) {
	return datastore.CreateUserToken(dbName, tok)
}
