package backend_test

import (
	"fmt"
	"log"
	"time"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/model"
)

type EntityDemo struct {
	ID        string `json:"id"`
	AccountID string `json:"accountId"`
	Name      string `json:"name"`
	Status    string `json:"status"`
}

func (x EntityDemo) String() string {
	return fmt.Sprintf("%s | %s", x.Name, x.Status)
}

func Example() {
	// we initiate config.Current as type config.AppConfig
	// using the in-memory database engine.
	// You'd use PostgreSQL or Mongo in your real configuration

	// Also note that the config package has a LoadConfig() function that loads
	// config from environment variables i.e.:
	// config.Current = LoadConfig()
	config.Current = config.AppConfig{
		AppEnv:           "dev",
		Port:             "8099",
		DatabaseURL:      "mem",
		DataStore:        "mem",
		LocalStorageURL:  "http://localhost:8099",
		NoFullTextSearch: true,
	}

	// the Setup function will initialize all services based on config
	backend.Setup(config.Current)

	// StaticBackend is multi-tenant by default, so you'll minimaly need
	// at least one Tenant with their Database for your app
	//
	// In a real application you need to decide if your customers will
	// have their own Database (multi-tenant) or not.
	cus := model.Tenant{
		Email:    "new@tenant.com",
		IsActive: true,
		Created:  time.Now(),
	}
	cus, err := backend.DB.CreateTenant(cus)
	if err != nil {
		fmt.Println(err)
		return
	}

	base := model.DatabaseConfig{
		TenantID: cus.ID,
		Name:     "random-name-here",
		IsActive: true,
		Created:  time.Now(),
	}
	base, err = backend.DB.CreateDatabase(base)
	if err != nil {
		fmt.Println(err)
		return
	}

	// let's create a user in this new Database
	// You'll need to create an model.Account and model.User for each of
	// your users. They'll need a session token to authenticate.
	usr := backend.Membership(base)

	// Role 100 is for root user, root user is your app's super user.
	// As the builder of your application you have a special user which can
	// execute things on behalf of other users. This is very useful on
	// background tasks were your app does not have the user's session token.
	sessionToken, user, err := usr.CreateAccountAndUser("user1@mail.com", "passwd123456", 100)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(len(sessionToken) > 10)

	// In a real application, you'd store the session token for this user
	// inside local storage and/or a cookie etc. On each request you'd
	// request this session token and authenticate this user via a middleware.

	// we simulate having authenticated this user (from middleware normally)
	auth := model.Auth{
		AccountID: user.AccountID,
		UserID:    user.ID,
		Email:     user.Email,
		Role:      user.Role,
		Token:     user.Token,
	}

	// this is what you'd normally do in your web handlers to execute a request

	// we create a ready to use CRUD and Query collection that's typed with
	// our EntityDemo. In your application you'd get a Collection for your own
	// type, for instance: Product, Order, Customer, Blog, etc.
	//
	// Notice how we're passing the auth: current user and base: current database
	// so the operations are made from the proper user and in the proper DB/Tenant.
	entities := backend.Collection[EntityDemo](auth, base, "entities")

	// once we have this collection, we can perform database operations
	newEntity := EntityDemo{Name: "Go example code", Status: "new"}

	newEntity, err = entities.Create(newEntity)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(newEntity)

	// the Create function returned our EntityDemo with the ID and AccountID
	// filled, we can now update this record.
	newEntity.Status = "updated"

	newEntity, err = entities.Update(newEntity.ID, newEntity)
	if err != nil {
		fmt.Println(err)
		return
	}

	// let's fetch this entity via its ID to make sure our changes have
	// been persisted.
	check, err := entities.GetByID(newEntity.ID)
	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println(check)
	// Output:
	// true
	// Go example code | new
	// Go example code | updated
}
