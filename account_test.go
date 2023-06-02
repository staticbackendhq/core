package staticbackend

import (
	"testing"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/model"
)

func TestUserAddRemoveFromAccount(t *testing.T) {
	u := model.Login{Email: "newuser@test.com", Password: "newuser1234"}
	resp := dbReq(t, acct.addUser, "POST", "/account/users", u)
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	// check if users is created
	users, err := backend.DB.ListUsers(pubKey, testAccountID)
	if err != nil {
		t.Fatal(err)
	}

	newUserID := ""
	for _, user := range users {
		if user.Email == "newuser@test.com" {
			newUserID = user.ID
			break
		}
	}

	if len(newUserID) == 0 {
		t.Fatal("unable to find new user")
	}

	resp2 := dbReq(t, acct.deleteUser, "DELETE", "/account/users/"+newUserID, nil)
	defer resp2.Body.Close()

	if resp2.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp2))
	}

	users, err = backend.DB.ListUsers(pubKey, testAccountID)
	if err != nil {
		t.Fatal(err)
	}

	for _, user := range users {
		if user.ID == newUserID {
			t.Fatal("deleted user was found?")
			break
		}
	}
}
