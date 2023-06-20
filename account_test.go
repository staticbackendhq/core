package staticbackend

import (
	"testing"
	"time"

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

	// adding user with same email should return an error
	resp2 := dbReq(t, acct.addUser, "POST", "/account/users", u)
	defer resp2.Body.Close()

	if resp2.StatusCode <= 299 {
		t.Fatal(GetResponseBody(t, resp2))
	}

	// check if users is created
	users, err := backend.DB.ListUsers(dbName, testAccountID)
	if err != nil {
		t.Fatal(err)
	}

	newUserID := ""
	for _, user := range users {
		if user.Email == "newuser@test.com" {
			newUserID = user.ID
			if !user.Created.After(time.Now().Add(-2 * time.Minute)) {
				t.Errorf("expected user to have a recent creation date, got %v", user.Created)
			}
			break
		}
	}

	if len(newUserID) == 0 {
		t.Fatal("unable to find new user")
	}

	resp3 := dbReq(t, acct.deleteUser, "DELETE", "/account/users/"+newUserID, nil)
	defer resp3.Body.Close()

	if resp3.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp3))
	}

	users, err = backend.DB.ListUsers(dbName, testAccountID)
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

func TestAddNewDatabase(t *testing.T) {
	resp := dbReq(t, acct.addDatabase, "GET", "/account/add-db", nil)
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}
}
