package memory

import (
	"fmt"
	"strings"

	"github.com/staticbackendhq/core/model"
)

func (m *Memory) FindUser(dbName, userID, token string) (tok model.User, err error) {
	if err = getByID(m, dbName, "sb_tokens", userID, &tok); err != nil {
		return
	} else if tok.Token != token {
		err = fmt.Errorf("token does not match")
	}
	return
}

func (m *Memory) FindRootUser(dbName, userID, accountID, token string) (tok model.User, err error) {
	tok, err = m.FindUser(dbName, userID, token)
	if err != nil {
		return
	} else if tok.AccountID != accountID {
		err = fmt.Errorf("invalid account id")
	}
	return
}

func (m *Memory) GetRootForBase(dbName string) (tok model.User, err error) {
	tokens, err := all[model.User](m, dbName, "sb_tokens")
	if err != nil {
		return
	}

	rootTokens := filter(tokens, func(t model.User) bool {
		return t.Role == 100
	})

	if len(rootTokens) == 0 {
		err = fmt.Errorf("cannot find root token")
		return
	}

	tok = rootTokens[0]
	return
}

func (m *Memory) FindUserByEmail(dbName, email string) (tok model.User, err error) {
	tokens, err := all[model.User](m, dbName, "sb_tokens")
	if err != nil {
		return
	}

	matches := filter(tokens, func(t model.User) bool {
		return strings.EqualFold(t.Email, email)
	})

	if len(matches) == 0 {
		err = fmt.Errorf("cannot find token by email")
		return
	}

	tok = matches[0]
	return
}

func (m *Memory) UserEmailExists(dbName, email string) (exists bool, err error) {
	if _, err := m.FindUserByEmail(dbName, email); err == nil {
		return true, nil
	}
	return
}

func (m *Memory) GetFirstUserFromAccountID(dbName, accountID string) (tok model.User, err error) {
	tokens, err := all[model.User](m, dbName, "sb_tokens")
	if err != nil {
		return
	}

	matches := filter(tokens, func(t model.User) bool {
		return t.AccountID == accountID
	})

	matches = sortSlice(matches, func(a, b model.User) bool {
		return a.Created.Before(b.Created)
	})

	if len(matches) == 0 {
		err = fmt.Errorf("no token found for this account")
		return
	}

	tok = matches[0]
	return
}
