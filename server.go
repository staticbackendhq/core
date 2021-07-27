package staticbackend

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"staticbackend/internal"
	"staticbackend/middleware"
	"staticbackend/realtime"
	"time"

	"staticbackend/cache"

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
	client   *mongo.Client
	volatile *cache.Cache
	AppEnv   = os.Getenv("APP_ENV")
)

// Start starts the web server and all dependencies services
func Start() {
	stripe.Key = os.Getenv("STRIPE_KEY")

	dbHost := flag.String("host", "localhost", "Hostname for mongodb")
	port := flag.String("port", "8099", "HTTP port to listen on")
	flag.Parse()

	if err := openDatabase(*dbHost); err != nil {
		log.Fatal(err)
	}

	volatile = cache.NewCache()

	// websockets
	hub := newHub(volatile)
	go hub.run()

	// Server Send Event, alternative to websocket
	b := realtime.NewBroker(func(ctx context.Context, key string) (string, error) {
		if _, err := middleware.ValidateAuthKey(client, ctx, key); err != nil {
			return "", err
		}
		return key, nil
	})

	database := &Database{
		client: client,
		cache:  volatile,
	}

	pubWithDB := []middleware.Middleware{
		middleware.Cors(),
		middleware.WithDB(client),
	}

	stdAuth := []middleware.Middleware{
		middleware.Cors(),
		middleware.WithDB(client),
		middleware.RequireAuth(client),
	}

	stdRoot := []middleware.Middleware{
		middleware.WithDB(client),
		middleware.RequireRoot(client),
	}

	http.Handle("/login", middleware.Chain(http.HandlerFunc(login), pubWithDB...))
	http.Handle("/register", middleware.Chain(http.HandlerFunc(register), pubWithDB...))
	http.Handle("/email", middleware.Chain(http.HandlerFunc(emailExists), pubWithDB...))
	//http.Handle("/setrole", chain(http.HandlerFunc(setRole), withDB))

	http.Handle("/sudogettoken/", middleware.Chain(http.HandlerFunc(sudoGetTokenFromAccountID), stdRoot...))

	// database routes
	http.Handle("/db/", middleware.Chain(http.HandlerFunc(database.dbreq), stdAuth...))
	http.Handle("/query/", middleware.Chain(http.HandlerFunc(database.query), stdAuth...))
	http.Handle("/sudoquery/", middleware.Chain(http.HandlerFunc(database.query), stdRoot...))
	http.Handle("/sudolistall/", middleware.Chain(http.HandlerFunc(database.listCollections), stdRoot...))
	http.Handle("/sudo/", middleware.Chain(http.HandlerFunc(database.dbreq), stdRoot...))
	http.Handle("/newid", middleware.Chain(http.HandlerFunc(database.newID), stdAuth...))

	// forms routes
	http.Handle("/postform/", middleware.Chain(http.HandlerFunc(submitForm), pubWithDB...))
	http.Handle("/form", middleware.Chain(http.HandlerFunc(listForm), stdRoot...))

	// storage
	http.Handle("/storage/upload", middleware.Chain(http.HandlerFunc(upload), stdAuth...))
	http.Handle("/sudostorage/delete", middleware.Chain(http.HandlerFunc(deleteFile), stdRoot...))

	// sudo actions
	http.Handle("/sudo/sendmail", middleware.Chain(http.HandlerFunc(sudoSendMail), stdRoot...))
	http.Handle("/sudo/cache", middleware.Chain(http.HandlerFunc(sudoCache), stdRoot...))

	// account
	acct := &accounts{}
	http.Handle("/account/init", middleware.Chain(http.HandlerFunc(acct.create), pubWithDB...))
	http.Handle("/account/auth", middleware.Chain(http.HandlerFunc(acct.auth), stdRoot...))
	http.Handle("/account/portal", middleware.Chain(http.HandlerFunc(acct.portal), stdRoot...))

	// stripe webhooks
	swh := stripeWebhook{}
	http.HandleFunc("/stripe", swh.process)

	http.HandleFunc("/ping", ping)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	http.Handle("/sse/connect", middleware.Chain(http.HandlerFunc(b.Accept), pubWithDB...))
	receiveMessage := func(w http.ResponseWriter, r *http.Request) {
		var msg internal.Command
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		b.Broadcast <- msg

		respond(w, http.StatusOK, true)
	}
	http.Handle("/sse/msg", middleware.Chain(http.HandlerFunc(receiveMessage), pubWithDB...))

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
		val, err := volatile.Get(key)
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

		if err := volatile.Set(data.Key, data.Value); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		respond(w, http.StatusOK, true)
	}
}
