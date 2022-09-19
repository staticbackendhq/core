package postgresql

import (
	"testing"
	"time"

	"github.com/staticbackendhq/core/model"
)

func TestCreateUserAccountAndToken(t *testing.T) {
	acctID, err := datastore.CreateAccount(confDBName, "unit@test.com")
	if err != nil {
		t.Fatal(err)
	}

	tok := model.User{
		AccountID: acctID,
		Token:     "123",
		Email:     "unit@test.com",
		Password:  "4321",
		Role:      0,
		Created:   time.Now(),
	}

	tokID, err := datastore.CreateUser(confDBName, tok)
	if err != nil {
		t.Fatal(err)
	} else if len(tokID) < 10 {
		t.Errorf("expected id to be len > 10 got %s", tokID)
	}
}

func TestGetFirstTokenFromAccountID(t *testing.T) {
	tok, err := datastore.GetFirstUserFromAccountID(confDBName, adminToken.AccountID)
	if err != nil {
		t.Fatal(err)
	} else if tok.ID != adminToken.ID {
		t.Errorf("wrong token, expected %s got %s", adminToken.ID, tok.ID)
	}
}

func TestSetPasswordResetCode(t *testing.T) {
	expected := "from_unit_test"

	if err := datastore.SetPasswordResetCode(confDBName, adminToken.ID, expected); err != nil {
		t.Fatal(err)
	}

	tok, err := datastore.FindUser(confDBName, adminToken.ID, adminToken.Token)
	if err != nil {
		t.Fatal(err)
	} else if tok.ResetCode != expected {
		t.Errorf("expected reset code to be %s got %s", expected, tok.ResetCode)
	}

	newpw := "changed_from_test"
	if err := datastore.ResetPassword(confDBName, adminEmail, expected, newpw); err != nil {
		t.Fatal(err)
	}

	tok2, err := datastore.FindUser(confDBName, adminToken.ID, adminToken.Token)
	if err != nil {
		t.Fatal(err)
	} else if tok2.Password != newpw {
		t.Errorf("expected password to be %s got %s", newpw, tok2.Password)
	}
}

func TestSetUserRole(t *testing.T) {
	newTok := model.User{
		AccountID: adminAccount.ID,
		Token:     "normal-user-token",
		Email:     "normal@test.com",
		Password:  "normal",
		Role:      1,
		ResetCode: "none",
		Created:   time.Now(),
	}

	newID, err := datastore.CreateUser(confDBName, newTok)
	if err != nil {
		t.Fatal(err)
	}

	if err := datastore.SetUserRole(confDBName, newTok.Email, 90); err != nil {
		t.Fatal(err)
	}

	tok, err := datastore.FindUser(confDBName, newID, newTok.Token)
	if err != nil {
		t.Fatal(err)
	} else if tok.Role != 90 {
		t.Errorf("expected role to be 90 got %d", tok.Role)
	}
}

func TestUserSetPassword(t *testing.T) {
	expected := "pw_changed"
	if err := datastore.UserSetPassword(confDBName, adminToken.ID, expected); err != nil {
		t.Fatal(err)
	}

	tok, err := datastore.FindUser(confDBName, adminToken.ID, adminToken.Token)
	if err != nil {
		t.Fatal(err)
	} else if tok.Password != expected {
		t.Errorf("expected password to be %s got %s", expected, tok.Password)
	}
}
