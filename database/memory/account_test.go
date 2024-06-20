package memory

import (
	"testing"

	"github.com/staticbackendhq/core/database/dbtest"
)

func TestFindToken(t *testing.T) {
	dbtest.FindToken(t, datastore, adminToken)
}

func TestFindRootToken(t *testing.T) {
	dbtest.FindRootToken(t, datastore, adminToken)
}

func TestGetRootForBase(t *testing.T) {
	dbtest.GetRootForBase(t, datastore, adminToken)
}

func TestFindTokenByEmail(t *testing.T) {
	dbtest.FindTokenByEmail(t, datastore, adminToken)
}

func TestUserEmailExists(t *testing.T) {
	dbtest.UserEmailExists(t, datastore)
}

func TestAccountList(t *testing.T) {
	dbtest.AccountList(t, datastore)
}
