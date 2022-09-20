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
	adminEmail    string
	adminPassword string
	adminAuth     model.Auth
	jwtToken      string

	base model.DatabaseConfig
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

	// initializes all core services basesd on config
	backend.Setup(cfg)

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
	cus := model.Tenant{
		Email:    adminEmail,
		IsActive: true,
		Created:  time.Now(),
	}

	cus, err := backend.DB.CreateTenant(cus)
	if err != nil {
		return err
	}

	base = model.DatabaseConfig{
		TenantID: cus.ID,
		Name:     "dev-memory-pk",
		IsActive: true,
	}

	base, err = backend.DB.CreateDatabase(base)
	if err != nil {
		return err
	}
	return nil
}

func createUser() error {
	mship := backend.Membership(base)
	jwt, user, err := mship.CreateAccountAndUser(adminEmail, adminPassword, 100)
	if err != nil {
		return err
	}

	adminAuth = model.Auth{
		AccountID: user.AccountID,
		UserID:    user.ID,
		Email:     adminEmail,
		Role:      100,
		Token:     user.Token,
	}

	jwtToken = string(jwt)

	return nil
}
