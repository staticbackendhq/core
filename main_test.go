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

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/database/mongo"
	"github.com/staticbackendhq/core/database/postgresql"
	"github.com/staticbackendhq/core/email"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/storage"
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

	database *Database
)

func TestMain(m *testing.M) {
	config.Current = config.LoadConfig()

	volatile = cache.NewCache()

	storer = storage.Local{}

	if strings.EqualFold(config.Current.DataStore, "mongo") {
		cl, err := openMongoDatabase("mongodb://localhost:27017")
		if err != nil {
			log.Fatal(err)
		}

		datastore = mongo.New(cl, volatile.PublishDocument)
	} else {
		dbConn, err := openPGDatabase("user=postgres password=postgres dbname=postgres sslmode=disable")
		if err != nil {
			log.Fatal(err)
		}

		datastore = postgresql.New(dbConn, volatile.PublishDocument, "./sql/")
	}

	database = &Database{cache: volatile}

	mp := config.Current.MailProvider
	if strings.EqualFold(mp, internal.MailProviderSES) {
		emailer = email.AWSSES{}
	} else {
		emailer = email.Dev{}
	}

	deleteAndSetupTestAccount()

	hub := newHub(volatile)
	go hub.run()

	ws := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	}))
	defer ws.Close()

	wsURL = "ws" + strings.TrimPrefix(ws.URL, "http")

	funexec = &functions{datastore: datastore, dbName: dbName}

	extexec = &extras{}

	os.Exit(m.Run())
}

func deleteAndSetupTestAccount() {
	if err := datastore.DeleteCustomer(dbName, admEmail); err != nil {
		log.Fatal(err)
	}

	cus := internal.Customer{
		Email: admEmail,
	}
	cus, err := datastore.CreateCustomer(cus)
	if err != nil {
		log.Fatal(err)
	}

	base := internal.BaseConfig{
		CustomerID:    cus.ID,
		Name:          dbName,
		AllowedDomain: []string{"localhost"},
		IsActive:      true,
		Created:       time.Now(),
	}

	base, err = datastore.CreateBase(base)
	if err != nil {
		log.Fatal(err)
	}

	pubKey = base.ID

	m := &membership{volatile: volatile}
	token, dbToken, err := m.createAccountAndUser(dbName, admEmail, password, 100)
	if err != nil {
		log.Fatal(err)
	}

	adminToken = string(token)

	rootToken = fmt.Sprintf("%s|%s|%s", dbToken.ID, dbToken.AccountID, dbToken.Token)

	token, _, err = m.createUser(dbName, dbToken.AccountID, userEmail, userPassword, 0)
	if err != nil {
		log.Fatal(err)
	}

	userToken = string(token)
}
