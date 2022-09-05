package backend_test

import (
	"os"
	"testing"
	"time"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/internal"
)

var (
	bkn backend.Backend

	adminEmail    string
	adminPassword string
	adminAuth     internal.Auth

	base internal.BaseConfig
)

func TestMain(t *testing.M) {
	// this simulate how a Go program would import
	// the backend package

	cfg := config.AppConfig{
		AppEnv:          "dev",
		FromCLI:         "yes",
		Port:            "8099",
		DatabaseURL:     "mem",
		DataStore:       "mem",
		LocalStorageURL: "http://localhost:8099",
	}

	bkn = backend.New(cfg)

	setup()

	os.Exit(t.Run())
}

func setup() {
	if err := createTenantAndDatabase(); err != nil {
		backend.Log.Fatal().Err(err)
	}

	if err := createUser(); err != nil {
		backend.Log.Fatal().Err(err)
	}
}

func createTenantAndDatabase() error {
	cus := internal.Customer{
		Email:    adminEmail,
		IsActive: true,
		Created:  time.Now(),
	}

	cus, err := bkn.Tenant.CreateCustomer(cus)
	if err != nil {
		return err
	}

	base = internal.BaseConfig{
		CustomerID: cus.ID,
		Name:       "dev-memory-pk",
		IsActive:   true,
	}

	base, err = bkn.Tenant.CreateBase(base)
	if err != nil {
		return err
	}
	return nil
}

func createUser() error {
	id, err := bkn.User.CreateAccount(base.Name, adminEmail)
	if err != nil {
		return err
	}

	tok := internal.Token{
		AccountID: id,
		Token:     backend.NewID(),
		Email:     adminEmail,
		Password:  adminPassword,
		Role:      100,
		Created:   time.Now(),
	}

	userID, err := bkn.User.CreateUserToken(base.Name, tok)
	if err != nil {
		return err
	}

	adminAuth = internal.Auth{
		AccountID: id,
		UserID:    userID,
		Email:     adminEmail,
		Role:      100,
		Token:     tok.Token,
	}

	return nil
}
