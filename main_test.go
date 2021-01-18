package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	dbName       = "unittest"
	email        = "unit@test.com"
	password     = "my_unittest_pw"
	userEmail    = "user@test.com"
	userPassword = "another_fake_password"
)

var (
	database   *Database
	wsURL      string
	pubKey     string
	adminToken string
	userToken  string
	rootToken  string
)

func TestMain(m *testing.M) {
	if err := openDatabase("localhost"); err != nil {
		log.Fatal(err)
	}

	deleteAndSetupTestAccount()

	cache := NewCache()

	hub := newHub(cache)
	go hub.run()

	ws := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	}))
	defer ws.Close()

	wsURL = "ws" + strings.TrimPrefix(ws.URL, "http")

	database = &Database{
		client: client,
		cache:  cache,
	}

	os.Exit(m.Run())
}

func deleteAndSetupTestAccount() {
	ctx := context.Background()

	if err := client.Database(dbName).Drop(ctx); err != nil {
		log.Fatal(err)
	}

	sysDB := client.Database("sbsys")

	if err := sysDB.Collection("accounts").Drop(ctx); err != nil {
		log.Fatal(err)
	}

	if err := sysDB.Collection("bases").Drop(ctx); err != nil {
		log.Fatal(err)
	}

	acctID := primitive.NewObjectID()
	cus := Customer{
		ID:    acctID,
		Email: email,
	}

	if _, err := sysDB.Collection("accounts").InsertOne(ctx, cus); err != nil {
		log.Fatal(err)
	}

	base := BaseConfig{
		ID:        primitive.NewObjectID(),
		SBID:      acctID,
		Name:      dbName,
		Whitelist: []string{"localhost"},
		Valid:     true,
	}

	if _, err := sysDB.Collection("bases").InsertOne(ctx, base); err != nil {
		log.Fatal(err)
	}

	pubKey = base.ID.Hex()

	db := client.Database(dbName)
	token, dbToken, err := createAccountAndUser(db, email, password, 100)
	if err != nil {
		log.Fatal(err)
	}

	adminToken = string(token)

	rootToken = fmt.Sprintf("%s|%s|%s", dbToken.ID.Hex(), dbToken.AccountID.Hex(), dbToken.Token)

	token, _, err = createUser(db, dbToken.AccountID, userEmail, userPassword, 0)
	if err != nil {
		log.Fatal(err)
	}

	userToken = string(token)
}
