package postgresql

import (
	"database/sql"
	"errors"
	"io"
	"log"
	"os"
	"path"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/spf13/afero"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/model"
)

const (
	adminEmail    = "pg@test.com"
	adminPassword = "test1234!"
	confDBName    = "testdb"
	colName       = "tasks"
)

var (
	datastore    *PostgreSQL
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

	migrationPath = "./sql/"
	appFS = afero.NewMemMapFs()

	if err := copyMigrationsToFS(); err != nil {
		log.Fatal(err)
	}

	dbConn, err := sql.Open("postgres", "user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer dbConn.Close()

	datastore = &PostgreSQL{DB: dbConn, PublishDocument: fakePubDocEvent}

	if err := datastore.Ping(); err != nil {
		log.Fatal(err)
	}

	// delete "sb" schema if exists
	_, err = dbConn.Exec("DROP SCHEMA IF EXISTS sb CASCADE;")
	if err != nil {
		log.Fatal(err)
	}

	if err := migrate(dbConn); err != nil {
		log.Fatal(err)
	}

	if err := datastore.DeleteTenant(confDBName, adminEmail); err != nil {
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
		return errors.New("testdb schema exists")
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

func copyMigrationsToFS() error {
	if err := appFS.Mkdir("sql", 0664); err != nil {
		return err
	}

	files, err := os.ReadDir("../../sql/")
	if err != nil {
		return err
	}

	for _, file := range files {
		f, err := appFS.Create("./sql/" + file.Name())
		if err != nil {
			return err
		}
		defer f.Close()

		src, err := os.Open(path.Join("../../sql", file.Name()))
		if err != nil {
			return err
		}
		defer src.Close()

		if n, err := io.Copy(f, src); err != nil {
			return err
		} else if n == 0 {
			return errors.New("unable to copy migration file")
		}
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
