package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/stripe/stripe-go/v71"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	client *mongo.Client
)

func main() {
	stripe.Key = os.Getenv("STRIPE_KEY")

	dbHost := flag.String("host", "localhost", "Hostname for mongodb")
	port := flag.String("port", "8099", "HTTP port to listen on")
	flag.Parse()

	if err := openDatabase(*dbHost); err != nil {
		log.Fatal(err)
	}

	cache := NewCache()

	// websockets
	hub := newHub(cache)
	go hub.run()

	database := &Database{
		client: client,
		cache:  cache,
	}

	http.Handle("/login", chain(http.HandlerFunc(login), withDB, cors))
	http.Handle("/register", chain(http.HandlerFunc(register), withDB, cors))
	http.Handle("/email", chain(http.HandlerFunc(emailExists), withDB, cors))
	http.Handle("/setrole", chain(http.HandlerFunc(setRole), withDB))

	// database routes
	http.Handle("/db/", chain(http.HandlerFunc(database.dbreq), auth, withDB, cors))
	http.Handle("/query/", chain(http.HandlerFunc(database.query), auth, withDB, cors))
	http.Handle("/sudoquery/", chain(http.HandlerFunc(database.query), requireRoot, withDB, cors))
	http.Handle("/sudo/", chain(http.HandlerFunc(database.dbreq), requireRoot, withDB, cors))
	http.Handle("/newid", chain(http.HandlerFunc(database.newID), auth, withDB, cors))

	// forms routes
	http.Handle("/postform/", chain(http.HandlerFunc(submitForm), withDB, cors))
	http.Handle("/form", chain(http.HandlerFunc(listForm), requireRoot, withDB, cors))

	// storage
	http.Handle("/storage/upload", chain(http.HandlerFunc(upload), auth, withDB, cors))

	// account
	acct := &accounts{}
	http.Handle("/account/init", chain(http.HandlerFunc(acct.create), cors))
	http.Handle("/account/auth", chain(http.HandlerFunc(acct.auth), requireRoot, withDB, cors))
	http.Handle("/account/portal", chain(http.HandlerFunc(acct.portal), requireRoot, withDB, cors))

	// stripe webhooks
	swh := stripeWebhook{}
	http.HandleFunc("/stripe", swh.process)

	http.HandleFunc("/ping", ping)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	log.Fatal(http.ListenAndServe(":"+*port, nil))
}

func openDatabase(dbHost string) error {
	uri := fmt.Sprintf("mongodb://app:%s@cluster0-shard-00-00.hq9ta.mongodb.net:27017,cluster0-shard-00-01.hq9ta.mongodb.net:27017,cluster0-shard-00-02.hq9ta.mongodb.net:27017/?ssl=true&replicaSet=atlas-xty9tn-shard-0&authSource=admin&retryWrites=true&w=majority", dbHost)
	if dbHost == "localhost" {
		uri = "mongodb://localhost:27017"
	}

	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	cl, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("cannot connect to mongo: %v", err)
	}

	if err := cl.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("Ping failed: %v", err)
	}

	client = cl
	return nil
}

type chainer func(h http.Handler) http.Handler

func chain(h http.Handler, middlewares ...chainer) http.Handler {
	next := h
	for _, m := range middlewares {
		next = m(next)
	}
	return next
}

func ping(w http.ResponseWriter, r *http.Request) {
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		http.Error(w, "connection failed to database, I'm down.", http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusOK, true)
}
