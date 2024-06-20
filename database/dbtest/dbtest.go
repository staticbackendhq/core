package dbtest

import (
	"testing"

	"github.com/staticbackendhq/core/database"
	"github.com/staticbackendhq/core/model"
)

const (
	adminEmail    = "pg@test.com"
	adminPassword = "test1234!"
	confDBName    = "testdb"
	colName       = "tasks"
)

func FindToken(t *testing.T, datastore database.Persister, adminToken model.User) {
	tok, err := datastore.FindUser(confDBName, adminToken.ID, adminToken.Token)
	if err != nil {
		t.Fatal(err)
	} else if tok.ID != adminToken.ID {
		t.Errorf("expected tok.id to be %s got %s", adminToken.ID, tok.ID)
	}
}

func FindRootToken(t *testing.T, datastore database.Persister, adminToken model.User) {
	tok, err := datastore.FindRootUser(confDBName, adminToken.ID, adminToken.AccountID, adminToken.Token)
	if err != nil {
		t.Fatal(err)
	} else if tok.ID != adminToken.ID {
		t.Errorf("expected token id to be %s got %s", adminToken.ID, tok.ID)
	}
}

func GetRootForBase(t *testing.T, datastore database.Persister, adminToken model.User) {
	tok, err := datastore.GetRootForBase(confDBName)
	if err != nil {
		t.Fatal(err)
	} else if tok.ID != adminToken.ID {
		t.Errorf("expected tok id to be %s got %s", adminToken.ID, tok.ID)
	}
}

func FindTokenByEmail(t *testing.T, datastore database.Persister, adminToken model.User) {
	tok, err := datastore.FindUserByEmail(confDBName, adminEmail)
	if err != nil {
		t.Fatal(err)
	} else if tok.ID != adminToken.ID {
		t.Errorf("expected tok id to be %s got %s", adminToken.ID, tok.ID)
	}
}

func UserEmailExists(t *testing.T, datastore database.Persister) {
	if exists, err := datastore.UserEmailExists(confDBName, adminEmail); err != nil {
		t.Fatal(err)
	} else if !exists {
		t.Errorf("email should exists")
	}
}

func AccountList(t *testing.T, datastore database.Persister) {
	accts, err := datastore.ListAccounts(confDBName)
	if err != nil {
		t.Fatal(err)
	} else if len(accts) == 0 {
		t.Errorf("expected at least 1 account, got %d", len(accts))
	}
}
