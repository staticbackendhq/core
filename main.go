package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"staticbackend/realtime"
	"time"

	"github.com/stripe/stripe-go/v71"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	AppEnvDev  = "dev"
	AppEnvProd = "prod"
)

var (
	client *mongo.Client
	cache  *Cache
	AppEnv = os.Getenv("APP_ENV")
)

func main() {
	stripe.Key = os.Getenv("STRIPE_KEY")

	dbHost := flag.String("host", "localhost", "Hostname for mongodb")
	port := flag.String("port", "8099", "HTTP port to listen on")
	flag.Parse()

	if err := openDatabase(*dbHost); err != nil {
		log.Fatal(err)
	}

	cache = NewCache()

	// websockets
	hub := newHub(cache)
	go hub.run()

	// Server Send Event, alternative to websocket
	b := realtime.NewBroker(func(ctx context.Context, key string) (string, error) {
		if _, err := validateAuthKey(ctx, key); err != nil {
			return "", err
		}
		return key, nil
	})

	database := &Database{
		client: client,
		cache:  cache,
	}

	http.Handle("/login", chain(http.HandlerFunc(login), withDB, cors))
	http.Handle("/register", chain(http.HandlerFunc(register), withDB, cors))
	http.Handle("/email", chain(http.HandlerFunc(emailExists), withDB, cors))
	http.Handle("/setrole", chain(http.HandlerFunc(setRole), withDB))

	http.Handle("/sudogettoken/", chain(http.HandlerFunc(sudoGetTokenFromAccountID), requireRoot, withDB))

	// database routes
	http.Handle("/db/", chain(http.HandlerFunc(database.dbreq), auth, withDB, cors))
	http.Handle("/query/", chain(http.HandlerFunc(database.query), auth, withDB, cors))
	http.Handle("/sudoquery/", chain(http.HandlerFunc(database.query), requireRoot, withDB, cors))
	http.Handle("/sudolistall/", chain(http.HandlerFunc(database.listCollections), requireRoot, withDB, cors))
	http.Handle("/sudo/", chain(http.HandlerFunc(database.dbreq), requireRoot, withDB, cors))
	http.Handle("/newid", chain(http.HandlerFunc(database.newID), auth, withDB, cors))

	// forms routes
	http.Handle("/postform/", chain(http.HandlerFunc(submitForm), withDB, cors))
	http.Handle("/form", chain(http.HandlerFunc(listForm), requireRoot, withDB, cors))

	// storage
	http.Handle("/storage/upload", chain(http.HandlerFunc(upload), auth, withDB, cors))
	http.Handle("/sudostorage/delete", chain(http.HandlerFunc(deleteFile), requireRoot, withDB))

	// sudo actions
	http.Handle("/sudo/sendmail", chain(http.HandlerFunc(sudoSendMail), requireRoot, withDB))
	http.Handle("/sudo/cache", chain(http.HandlerFunc(sudoCache), requireRoot))

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

	http.Handle("/sse/connect", chain(http.HandlerFunc(b.Accept), withDB, cors))
	receiveMessage := func(w http.ResponseWriter, r *http.Request) {
		var msg realtime.Command
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		b.Broadcast <- msg

		respond(w, http.StatusOK, true)
	}
	http.Handle("/sse/msg", chain(http.HandlerFunc(receiveMessage), cors))

	log.Fatal(http.ListenAndServe(":"+*port, nil))
}

func openDatabase(dbHost string) error {
	uri := os.Getenv("DATABASE_URL")
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

func sudoCache(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		key := r.URL.Query().Get("key")
		val, err := cache.Get(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		respond(w, http.StatusOK, val)
	} else if r.Method == http.MethodPost {
		data := new(struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		})
		if err := parseBody(r.Body, &data); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := cache.Set(data.Key, data.Value); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		respond(w, http.StatusOK, true)
	}
}
