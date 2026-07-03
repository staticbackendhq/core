package staticbackend

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/model"
)

func TestUserAddRemoveFromAccount(t *testing.T) {
	u := model.Login{Email: "newuser@test.com", Password: "newuser1234"}
	resp := dbReq(t, acct.addUser, "POST", "/account/users", u)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	// adding user with same email should return an error
	resp2 := dbReq(t, acct.addUser, "POST", "/account/users", u)
	defer func() { _ = resp2.Body.Close() }()

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
			if user.Created.Format("2006-01-02") != time.Now().Format("2006-01-02") {
				t.Errorf("expected user to have a recent creation date, got %v", user.Created)
			}
			break
		}
	}

	if len(newUserID) == 0 {
		t.Fatal("unable to find new user")
	}

	resp3 := dbReq(t, acct.deleteUser, "DELETE", "/account/users/"+newUserID, nil)
	defer func() { _ = resp3.Body.Close() }()

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

func TestDeleteUserAllowsEqualRole50(t *testing.T) {
	conf, err := backend.DB.FindDatabase(pubKey)
	if err != nil {
		t.Fatal(err)
	}

	adminToken, admin, err := backend.Membership(conf).CreateUser(testAccountID, "delete-admin-50@test.com", userPassword, 50)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = backend.DB.RemoveUser(model.Auth{AccountID: testAccountID, UserID: admin.ID, Role: 100}, dbName, admin.ID)
	})

	_, target, err := backend.Membership(conf).CreateUser(testAccountID, "delete-target-50@test.com", userPassword, 50)
	if err != nil {
		t.Fatal(err)
	}

	resp := authReqWithToken(t, string(adminToken), acct.deleteUser, "DELETE", "/account/users/"+target.ID, nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	if _, err := backend.DB.GetUserByID(dbName, testAccountID, target.ID); err == nil {
		t.Fatal("expected role 50 target user to be deleted")
	}
}

func TestDeleteUserRequiresRole50(t *testing.T) {
	conf, err := backend.DB.FindDatabase(pubKey)
	if err != nil {
		t.Fatal(err)
	}

	roleZeroToken, roleZeroUser, err := backend.Membership(conf).CreateUser(testAccountID, "delete-role-0@test.com", userPassword, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = backend.DB.RemoveUser(model.Auth{AccountID: testAccountID, UserID: roleZeroUser.ID, Role: 100}, dbName, roleZeroUser.ID)
	})

	_, target, err := backend.Membership(conf).CreateUser(testAccountID, "delete-target-role-0@test.com", userPassword, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = backend.DB.RemoveUser(model.Auth{AccountID: testAccountID, UserID: target.ID, Role: 100}, dbName, target.ID)
	})

	resp := authReqWithToken(t, string(roleZeroToken), acct.deleteUser, "DELETE", "/account/users/"+target.ID, nil)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected status %d got %d: %s", http.StatusUnauthorized, resp.StatusCode, GetResponseBody(t, resp))
	}
}

func TestListUsersByRole(t *testing.T) {
	roleEmail := "role-filter-user@test.com"
	_, roleUser, err := backend.Membership(model.DatabaseConfig{Name: dbName}).CreateUser(testAccountID, roleEmail, userPassword, 50)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = backend.DB.RemoveUser(model.Auth{AccountID: testAccountID, UserID: roleUser.ID, Role: 100}, dbName, roleUser.ID)
	})

	resp := dbReq(t, acct.addUser, "GET", "/account/users?role=50", nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	var users []model.User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		t.Fatal(err)
	}
	if len(users) == 0 {
		t.Fatal("expected at least one user")
	}
	for _, user := range users {
		if user.Role != 50 {
			t.Fatalf("expected only role 50 users, got role %d for %s", user.Role, user.Email)
		}
	}

	resp = dbReq(t, acct.addUser, "GET", "/account/users?role=bad", nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status %d got %d", http.StatusBadRequest, resp.StatusCode)
	}
}

func TestAddNewDatabase(t *testing.T) {
	resp := dbReq(t, acct.addDatabase, "GET", "/account/add-db", nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}
}

func TestListAssociations(t *testing.T) {
	resp := dbReq(t, acct.listAssociations, "GET", "/account/associations", nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	// result may be nil/null when there are no associations — that is valid
	var result []model.AccountUser
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal("decoding response:", err)
	}
}

func TestGetUserAccounts(t *testing.T) {
	resp := dbReq(t, acct.getUserAccounts, "GET", "/account/user-accounts?email="+admEmail, nil, true)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	type accountEntry struct {
		AccountID string `json:"accountId"`
		Role      int    `json:"role"`
		Home      bool   `json:"home"`
	}
	var result []accountEntry
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal("decoding response:", err)
	}
	if len(result) == 0 {
		t.Error("expected at least the home account entry")
	}

	found := false
	for _, entry := range result {
		if entry.Home {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected one entry with home=true")
	}
}

func TestGetUserAccountsMissingEmail(t *testing.T) {
	resp := dbReq(t, acct.getUserAccounts, "GET", "/account/user-accounts", nil, true)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 got %d", resp.StatusCode)
	}
}

func TestCrossAccountUserAssociation(t *testing.T) {
	// Create a user that lives in a *different* account so that addUser
	// triggers the cross-account association path instead of the
	// "email already in use in this account" rejection.
	const crossEmail = "cross-account-user@test.com"

	otherAcctID, err := backend.DB.CreateAccount(dbName, "cross-owner@test.com")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := backend.DB.CreateUser(dbName, model.User{
		AccountID: otherAcctID,
		Email:     crossEmail,
		Password:  "doesnotmatter",
		Token:     backend.DB.NewID(),
		Role:      0,
		Created:   time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	// addUser as the admin (testAccountID) with an email from a different account
	// should create a cross-account association, not a new user record
	resp := dbReq(t, acct.addUser, "POST", "/account/users", model.Login{Email: crossEmail})
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	var associatedUser model.User
	if err := json.NewDecoder(resp.Body).Decode(&associatedUser); err != nil {
		t.Fatal("decoding response:", err)
	}
	if len(associatedUser.Token) == 0 {
		t.Error("expected a non-empty association token")
	}
	if associatedUser.Email != crossEmail {
		t.Errorf("expected email %s got %s", crossEmail, associatedUser.Email)
	}

	resp = dbReq(t, acct.addUser, "GET", "/account/users", nil)
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	var users []model.User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		t.Fatal("decoding list response:", err)
	}

	foundAssociatedUser := false
	for _, user := range users {
		if user.ID == associatedUser.ID && user.AccountID == testAccountID && user.Email == crossEmail {
			foundAssociatedUser = true
			break
		}
	}
	if !foundAssociatedUser {
		t.Fatalf("expected associated user %s in account user list", crossEmail)
	}

	resp = dbReq(t, acct.addUser, "GET", "/account/users?role=0", nil)
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	users = nil
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		t.Fatal("decoding role-filtered list response:", err)
	}

	foundAssociatedUser = false
	for _, user := range users {
		if user.Role != 0 {
			t.Fatalf("expected only role 0 users, got role %d for %s", user.Role, user.Email)
		}
		if user.ID == associatedUser.ID && user.AccountID == testAccountID && user.Email == crossEmail {
			foundAssociatedUser = true
		}
	}
	if !foundAssociatedUser {
		t.Fatalf("expected associated role 0 user %s in filtered account user list", crossEmail)
	}

	resp = dbReq(t, acct.addUser, "GET", "/account/users?role=50", nil)
	defer resp.Body.Close()

	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	users = nil
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		t.Fatal("decoding role-filtered list response:", err)
	}

	for _, user := range users {
		if user.ID == associatedUser.ID && user.AccountID == testAccountID && user.Email == crossEmail {
			t.Fatalf("did not expect associated role 0 user %s in role 50 filtered account user list", crossEmail)
		}
	}

	// clean up
	assoc, err := backend.DB.GetAccountUser(dbName, associatedUser.ID, testAccountID)
	if err != nil {
		t.Fatal("finding association for cleanup:", err)
	}
	if err := backend.DB.DeleteAccountUser(dbName, assoc.ID); err != nil {
		t.Fatal("cleanup failed:", err)
	}
}
