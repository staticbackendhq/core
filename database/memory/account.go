package memory

import (
	"fmt"
	"strings"

	"github.com/staticbackendhq/core/internal"
)

func (m *Memory) FindToken(dbName, tokenID, token string) (tok internal.Token, err error) {
	if err = getByID[*internal.Token](m, dbName, "sb_tokens", tokenID, &tok); err != nil {
		return
	} else if tok.Token != token {
		err = fmt.Errorf("token does not match")
	}
	return
}

func (m *Memory) FindRootToken(dbName, tokenID, accountID, token string) (tok internal.Token, err error) {
	tok, err = m.FindToken(dbName, tokenID, token)
	if err != nil {
		return
	} else if tok.AccountID != accountID {
		err = fmt.Errorf("invalid account id")
	}
	return
}

func (m *Memory) GetRootForBase(dbName string) (tok internal.Token, err error) {
	tokens, err := all[internal.Token](m, dbName, "sb_tokens")
	if err != nil {
		return
	}

	rootTokens := filter[internal.Token](tokens, func(t internal.Token) bool {
		return t.Role == 100
	})

	if len(rootTokens) == 0 {
		err = fmt.Errorf("cannot find root token")
		return
	}

	tok = rootTokens[0]
	return
}

func (m *Memory) FindTokenByEmail(dbName, email string) (tok internal.Token, err error) {
	tokens, err := all[internal.Token](m, dbName, "sb_tokens")
	if err != nil {
		return
	}

	matches := filter[internal.Token](tokens, func(t internal.Token) bool {
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
	if _, err := m.FindTokenByEmail(dbName, email); err != nil {
		return true, nil
	}
	return
}

func (m *Memory) GetFirstTokenFromAccountID(dbName, accountID string) (tok internal.Token, err error) {
	tokens, err := all[internal.Token](m, dbName, "sb_tokens")
	if err != nil {
		return
	}

	matches := filter[internal.Token](tokens, func(t internal.Token) bool {
		return t.AccountID == accountID
	})

	matches = sortSlice[internal.Token](matches, func(a, b internal.Token) bool {
		return a.Created.Before(b.Created)
	})

	if len(matches) == 0 {
		err = fmt.Errorf("no token found for this account")
		return
	}

	tok = matches[0]
	return
}
