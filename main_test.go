package staticbackend

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/staticbackendhq/core/backend"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/model"
)

const (
	dbName       = "unittest"
	admEmail     = "allunit@test.com"
	password     = "my_unittest_pw"
	userEmail    = "user@test.com"
	userPassword = "another_fake_password"
)

var (
	extexec    *extras
	funexec    *functions
	wsURL      string
	pubKey     string
	adminToken string
	userToken  string
	rootToken  string

	mship *membership
	db    *Database
)

func TestMain(m *testing.M) {
	config.Current = config.LoadConfig()

	bkn = backend.New(config.Current)

	db = &Database{cache: backend.Cache, log: backend.Log}

	mship = &membership{log: backend.Log}

	deleteAndSetupTestAccount()

	hub := newHub(backend.Cache)
	go hub.run()

	ws := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveWs(backend.Log, hub, w, r)
	}))
	defer ws.Close()

	wsURL = "ws" + strings.TrimPrefix(ws.URL, "http")

	funexec = &functions{datastore: backend.DB, dbName: dbName}

	extexec = &extras{}

	os.Exit(m.Run())
}

func deleteAndSetupTestAccount() {
	if err := backend.DB.DeleteCustomer(dbName, admEmail); err != nil {
		log.Fatal(err)
	}

	cus := model.Customer{
		Email: admEmail,
	}
	cus, err := backend.DB.CreateCustomer(cus)
	if err != nil {
		log.Fatal(err)
	}

	base := model.BaseConfig{
		CustomerID:    cus.ID,
		Name:          dbName,
		AllowedDomain: []string{"localhost"},
		IsActive:      true,
		Created:       time.Now(),
	}

	base, err = backend.DB.CreateBase(base)
	if err != nil {
		log.Fatal(err)
	}

	pubKey = base.ID

	token, dbToken, err := mship.createAccountAndUser(dbName, admEmail, password, 100)
	if err != nil {
		log.Fatal(err)
	}

	adminToken = string(token)

	rootToken = fmt.Sprintf("%s|%s|%s", dbToken.ID, dbToken.AccountID, dbToken.Token)

	token, _, err = mship.createUser(dbName, dbToken.AccountID, userEmail, userPassword, 0)
	if err != nil {
		log.Fatal(err)
	}

	userToken = string(token)
}
