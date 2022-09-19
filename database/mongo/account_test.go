package mongo

import "testing"

func TestFindToken(t *testing.T) {
	tok, err := datastore.FindUser(confDBName, adminToken.ID, adminToken.Token)
	if err != nil {
		t.Fatal(err)
	} else if tok.ID != adminToken.ID {
		t.Errorf("expected tok.id to be %s got %s", adminToken.ID, tok.ID)
	}
}

func TestFindRootToken(t *testing.T) {
	tok, err := datastore.FindRootUser(confDBName, adminToken.ID, adminToken.AccountID, adminToken.Token)
	if err != nil {
		t.Fatal(err)
	} else if tok.ID != adminToken.ID {
		t.Errorf("expected token id to be %s got %s", adminToken.ID, tok.ID)
	}
}

func TestGetRootForBase(t *testing.T) {
	tok, err := datastore.GetRootForBase(confDBName)
	if err != nil {
		t.Fatal(err)
	} else if tok.ID != adminToken.ID {
		t.Errorf("expected tok id to be %s got %s", adminToken.ID, tok.ID)
	}
}

func TestFindTokenByEmail(t *testing.T) {
	tok, err := datastore.FindUserByEmail(confDBName, adminEmail)
	if err != nil {
		t.Fatal(err)
	} else if tok.ID != adminToken.ID {
		t.Errorf("expected tok id to be %s got %s", adminToken.ID, tok.ID)
	}
}

func TestUserEmailExists(t *testing.T) {
	if exists, err := datastore.UserEmailExists(confDBName, adminEmail); err != nil {
		t.Fatal(err)
	} else if !exists {
		t.Errorf("email should exists")
	}
}
