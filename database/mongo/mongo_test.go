package mongo

import (
	"context"
	"errors"
	"github.com/staticbackendhq/core/logger"
	"log"
	"os"
	"testing"
	"time"

	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/model"
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
	dbTest       model.DatabaseConfig
	adminAccount model.Account
	adminToken   model.User
	adminAuth    model.Auth
)

func fakePubDocEvent(channel, typ string, v interface{}) {
	//no event pub in those tests
}

func TestMain(m *testing.M) {
	config.Current = config.LoadConfig()

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

	datastore = &Mongo{
		Client:          cl,
		Ctx:             context.Background(),
		PublishDocument: fakePubDocEvent,
		log:             logger.Get(config.Current),
	}

	if err := datastore.Ping(); err != nil {
		log.Fatal(err)
	}

	if err := datastore.DeleteTenant(confDBName, adminEmail); err != nil {
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

	cus := model.Tenant{
		Email:          adminEmail,
		StripeID:       adminEmail,
		SubscriptionID: adminEmail,
		IsActive:       false, // will be turn true via TestActivateCustomer
		Created:        time.Now(),
	}

	cus, err = datastore.CreateTenant(cus)
	if err != nil {
		return err
	}

	base := model.DatabaseConfig{
		TenantID:      cus.ID,
		Name:          confDBName,
		AllowedDomain: []string{"localhost"},
		IsActive:      true,
		Created:       time.Now(),
	}

	exists, err = datastore.DatabaseExists(confDBName)
	if exists {
		return errors.New("testdb db exists")
	}

	base, err = datastore.CreateDatabase(base)
	if err != nil {
		return err
	}

	dbTest = base

	return nil
}

func createAdminAccountAndToken() error {
	acctID, err := datastore.CreateAccount(confDBName, adminEmail)
	if err != nil {
		return err
	}

	adminAccount = model.Account{ID: acctID, Email: adminEmail}

	adminToken = model.User{
		AccountID: adminAccount.ID,
		Token:     adminEmail,
		Email:     adminEmail,
		Password:  adminEmail,
		Role:      100,
		Created:   time.Now(),
	}

	tokID, err := datastore.CreateUser(confDBName, adminToken)
	if err != nil {
		return err
	}

	adminToken.ID = tokID

	adminAuth = model.Auth{
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
