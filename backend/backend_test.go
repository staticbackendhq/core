package backend_test

import (
	"os"
	"testing"
	"time"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/model"
)

var (
	bkn backend.Backend

	adminEmail    string
	adminPassword string
	adminAuth     model.Auth
	jwtToken      string

	base model.BaseConfig
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
	cus := model.Customer{
		Email:    adminEmail,
		IsActive: true,
		Created:  time.Now(),
	}

	cus, err := backend.DB.CreateCustomer(cus)
	if err != nil {
		return err
	}

	base = model.BaseConfig{
		CustomerID: cus.ID,
		Name:       "dev-memory-pk",
		IsActive:   true,
	}

	base, err = backend.DB.CreateBase(base)
	if err != nil {
		return err
	}
	return nil
}

func createUser() error {
	id, err := bkn.User(base.ID).CreateAccount(adminEmail)
	if err != nil {
		return err
	}

	userID, err := bkn.User(base.ID).CreateUserToken(id, adminEmail, adminPassword, 100)
	if err != nil {
		return err
	}

	tok := model.Token{
		ID:        userID,
		AccountID: id,
		Email:     adminEmail,
		Role:      100,
		Created:   time.Now(),
	}

	adminAuth = model.Auth{
		AccountID: id,
		UserID:    userID,
		Email:     adminEmail,
		Role:      100,
		Token:     tok.Token,
	}

	jwtToken, err = bkn.User(base.ID).Authenticate(adminEmail, adminPassword)
	if err != nil {
		return err
	}

	return nil
}
