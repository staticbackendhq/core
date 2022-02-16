package mongo

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/staticbackendhq/core/internal"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	adminEmail    = "pg@test.com"
	adminPassword = "test1234!"
	confDBName    = "testdb"
	colName       = "tasks"
)

var (
	datastore    *Mongo
	dbTest       internal.BaseConfig
	adminAccount internal.Account
	adminToken   internal.Token
	adminAuth    internal.Auth
)

func TestMain(m *testing.M) {
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	cl, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := cl.Disconnect(ctx); err != nil {
			log.Fatal(err)
		}
	}()

	datastore = &Mongo{Client: cl, Ctx: context.Background()}

	if err := datastore.Ping(); err != nil {
		log.Fatal(err)
	}

	if err := datastore.DeleteCustomer(confDBName, adminEmail); err != nil {
		log.Fatal(err)
	}

	if err := createCustomerAndDB(); err != nil {
		log.Fatal(err)
	}

	if err := createAdminAccountAndToken(); err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func createCustomerAndDB() error {
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
	if exists {
		return errors.New("testdb db exists")
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
