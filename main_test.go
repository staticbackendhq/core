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

	backend.Setup(config.Current)

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
	if err := backend.DB.DeleteTenant(dbName, admEmail); err != nil {
		log.Fatal(err)
	}

	cus := model.Tenant{
		Email: admEmail,
	}
	cus, err := backend.DB.CreateTenant(cus)
	if err != nil {
		log.Fatal(err)
	}

	base := model.DatabaseConfig{
		TenantID:      cus.ID,
		Name:          dbName,
		AllowedDomain: []string{"localhost"},
		IsActive:      true,
		Created:       time.Now(),
	}

	base, err = backend.DB.CreateDatabase(base)
	if err != nil {
		log.Fatal(err)
	}

	pubKey = base.ID

	usrSvc := backend.Membership(base)
	token, dbToken, err := usrSvc.CreateAccountAndUser(admEmail, password, 100)
	if err != nil {
		log.Fatal(err)
	}

	adminToken = string(token)

	rootToken = fmt.Sprintf("%s|%s|%s", dbToken.ID, dbToken.AccountID, dbToken.Token)

	token, _, err = usrSvc.CreateUser(dbToken.AccountID, userEmail, userPassword, 0)
	if err != nil {
		log.Fatal(err)
	}

	userToken = string(token)
}
