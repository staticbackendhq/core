package memory

import (
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/staticbackendhq/core/internal"
)

const (
	adminEmail    = "pg@test.com"
	adminPassword = "test1234!"
	confDBName    = "testdb"
	colName       = "tasks"
)

var (
	datastore    *Memory
	dbTest       internal.BaseConfig
	adminAccount internal.Account
	adminToken   internal.Token
	adminAuth    internal.Auth
)

func fakePubDocEvent(channel, typ string, v interface{}) {
	//no event pub in those tests
}

func TestMain(m *testing.M) {
	db := make(map[string]map[string][]byte)
	if err := initDB(db); err != nil {
		log.Fatal(err)
	}

	datastore = &Memory{DB: db, PublishDocument: fakePubDocEvent}

	if err := datastore.Ping(); err != nil {
		log.Fatal(err)
	}

	if err := datastore.DeleteCustomer(confDBName, adminEmail); err != nil {
		log.Fatal(err)
	}

	if err := createCustomerAndSchema(); err != nil {
		log.Fatal(err)
	}

	if err := createAdminAccountAndToken(); err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func createCustomerAndSchema() error {
	exists, err := datastore.EmailExists(adminEmail)
	if err != nil {
		return err
	} else if exists {
		return errors.New("admin email exists, should not")
	}

	cus := internal.Customer{
		Email:          adminEmail,
		StripeID:       adminEmail,
		SubscriptionID: adminEmail,
		IsActive:       false, // will be turn true via TestActivateCustomer
		Created:        time.Now(),
	}

	cus, err = datastore.CreateCustomer(cus)
	if err != nil {
		return err
	}

	base := internal.BaseConfig{
		CustomerID:    cus.ID,
		Name:          confDBName,
		AllowedDomain: []string{"localhost"},
		IsActive:      true,
		Created:       time.Now(),
	}

	exists, err = datastore.DatabaseExists(confDBName)
	if err != nil {
		return err
	}

	if exists {
		return errors.New("testdb schema exists")
	}

	base, err = datastore.CreateBase(base)
	if err != nil {
		return err
	}

	dbTest = base

	return nil
}

func createAdminAccountAndToken() error {
	acctID, err := datastore.CreateUserAccount(confDBName, adminEmail)
	if err != nil {
		return err
	}

	adminAccount = internal.Account{ID: acctID, Email: adminEmail}

	adminToken = internal.Token{
		AccountID: adminAccount.ID,
		Token:     adminEmail,
		Email:     adminEmail,
		Password:  adminEmail,
		Role:      100,
		Created:   time.Now(),
	}

	tokID, err := datastore.CreateUserToken(confDBName, adminToken)
	if err != nil {
		return err
	}

	adminToken.ID = tokID

	adminAuth = internal.Auth{
		AccountID: acctID,
		UserID:    tokID,
		Email:     adminEmail,
		Role:      100,
		Token:     adminToken.Token,
	}
	return nil
}

func TestCreateIndex(t *testing.T) {
	data := make(map[string]interface{})
	data["idxfield"] = "unit test"
	data["value"] = 123

	if _, err := datastore.CreateDocument(adminAuth, confDBName, "testindex", data); err != nil {
		t.Fatal(err)
	}

	if err := datastore.CreateIndex(confDBName, "testindex", "idxfield"); err != nil {
		t.Fatal(err)
	}
}
