package postgresql

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/staticbackendhq/core/internal"
)

const (
	adminEmail    = "pg@test.com"
	adminPassword = "test1234!"
	confDBName    = "testdb"
	colName       = "tasks"
)

var (
	datastore    *PostgreSQL
	dbTest       internal.BaseConfig
	adminAccount internal.Account
	adminToken   internal.Token
	adminAuth    internal.Auth
)

func TestMain(m *testing.M) {
	dbConn, err := sql.Open("postgres", "user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer dbConn.Close()

	if err := dbConn.Ping(); err != nil {
		log.Fatal(err)
	}

	if err := cleanupTestSchema(dbConn); err != nil {
		log.Fatal(err)
	}

	datastore = &PostgreSQL{DB: dbConn}

	if err := createCustomerAndSchema(); err != nil {
		log.Fatal(err)
	}

	if err := createAdminAccountAndToken(); err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func cleanupTestSchema(db *sql.DB) error {
	_, err := db.Exec(fmt.Sprintf(`DROP SCHEMA IF EXISTS %s CASCADE;`, confDBName))
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		DELETE FROM sb.customers WHERE email = $1;
	`, adminEmail)

	return err
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
