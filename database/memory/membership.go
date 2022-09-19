package memory

import (
	"fmt"
	"time"

	"github.com/staticbackendhq/core/model"
)

func (m *Memory) CreateAccount(dbName, email string) (id string, err error) {
	id = m.NewID()

	acct := model.Account{
		ID:      id,
		Created: time.Now(),
		Email:   email,
	}
	err = create(m, dbName, "sb_accounts", id, acct)
	return
}

func (m *Memory) CreateUser(dbName string, tok model.User) (id string, err error) {
	id = m.NewID()
	tok.ID = id

	err = create(m, dbName, "sb_tokens", id, tok)
	return
}

func (m *Memory) SetPasswordResetCode(dbName, tokenID, code string) error {
	var tok model.User
	if err := getByID(m, dbName, "sb_tokens", tokenID, &tok); err != nil {
		return err
	}

	tok.ResetCode = code
	return create(m, dbName, "sb_tokens", tok.ID, tok)
}

func (m *Memory) ResetPassword(dbName, email, code, password string) error {
	tok, err := m.FindUserByEmail(dbName, email)
	if err != nil {
		return err
	} else if tok.ResetCode != code {
		return fmt.Errorf("invalid code")
	}

	tok.Password = password
	return create(m, dbName, "sb_tokens", tok.ID, tok)
}

func (m *Memory) SetUserRole(dbName, email string, role int) error {
	tok, err := m.FindUserByEmail(dbName, email)
	if err != nil {
		return err
	}

	tok.Role = role
	return create(m, dbName, "sb_tokens", tok.ID, tok)
}

func (m *Memory) UserSetPassword(dbName, tokenID, password string) error {
	var tok model.User
	if err := getByID(m, dbName, "sb_tokens", tokenID, &tok); err != nil {
		return err
	}

	tok.Password = password
	return create(m, dbName, "sb_tokens", tok.ID, tok)
}
