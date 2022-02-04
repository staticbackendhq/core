package staticbackend

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/db"
	"github.com/staticbackendhq/core/internal"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	dbName       = "unittest"
	admEmail     = "unit@test.com"
	password     = "my_unittest_pw"
	userEmail    = "user@test.com"
	userPassword = "another_fake_password"
)

var (
	database   *Database
	funexec    *functions
	wsURL      string
	pubKey     string
	adminToken string
	userToken  string
	rootToken  string
)

func TestMain(m *testing.M) {
	if err := openDatabase("mongodb://localhost:27017"); err != nil {
		log.Fatal(err)
	}

	volatile = cache.NewCache()

	deleteAndSetupTestAccount()

	hub := newHub(volatile)
	go hub.run()

	ws := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	}))
	defer ws.Close()

	wsURL = "ws" + strings.TrimPrefix(ws.URL, "http")

	database = &Database{
		client: client,
		cache:  volatile,
		base:   &db.Base{PublishDocument: volatile.PublishDocument},
	}

	funexec = &functions{base: &db.Base{PublishDocument: volatile.PublishDocument}}

	os.Exit(m.Run())
}

func deleteAndSetupTestAccount() {
	ctx := context.Background()

	if err := client.Database(dbName).Drop(ctx); err != nil {
		log.Fatal(err)
	}

	sysDB := client.Database("sbsys")

	if _, err := sysDB.Collection("accounts").DeleteMany(ctx, bson.M{"email": admEmail}); err != nil {
		log.Fatal(err)
	}

	if _, err := sysDB.Collection("bases").DeleteOne(ctx, bson.M{"name": dbName}); err != nil {
		log.Fatal(err)
	}

	acctID := primitive.NewObjectID()
	cus := internal.Customer{
		ID:    acctID,
		Email: admEmail,
	}

	if _, err := sysDB.Collection("accounts").InsertOne(ctx, cus); err != nil {
		log.Fatal(err)
	}

	base := internal.BaseConfig{
		ID:        primitive.NewObjectID(),
		SBID:      acctID,
		Name:      dbName,
		Whitelist: []string{"localhost"},
		IsActive:  true,
	}

	if _, err := sysDB.Collection("bases").InsertOne(ctx, base); err != nil {
		log.Fatal(err)
	}

	pubKey = base.ID.Hex()

	m := &membership{volatile: volatile}

	db := client.Database(dbName)
	token, dbToken, err := m.createAccountAndUser(db, admEmail, password, 100)
	if err != nil {
		log.Fatal(err)
	}

	adminToken = string(token)

	rootToken = fmt.Sprintf("%s|%s|%s", dbToken.ID.Hex(), dbToken.AccountID.Hex(), dbToken.Token)

	token, _, err = m.createUser(db, dbToken.AccountID, userEmail, userPassword, 0)
	if err != nil {
		log.Fatal(err)
	}

	userToken = string(token)
}
