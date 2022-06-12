package staticbackend

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/database/memory"
	"github.com/staticbackendhq/core/database/mongo"
	"github.com/staticbackendhq/core/database/postgresql"
	"github.com/staticbackendhq/core/email"
	"github.com/staticbackendhq/core/function"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/middleware"
	"github.com/staticbackendhq/core/realtime"
	"github.com/staticbackendhq/core/storage"

	"github.com/stripe/stripe-go/v72"
	mongodrv "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"golang.org/x/sync/errgroup"

	_ "github.com/lib/pq"
)

const (
	AppEnvDev  = "dev"
	AppEnvProd = "prod"
)

var (
	datastore internal.Persister
	volatile  internal.Volatilizer
	emailer   internal.Mailer
	storer    internal.Storer
	AppEnv    = config.Current.AppEnv
)

// Start starts the web server and all dependencies services
func Start(c config.AppConfig) {
	config.Current = c

	stripe.Key = config.Current.StripeKey

	if err := loadTemplates(); err != nil {
		// if we're running from the CLI, no need to load templates
		if len(config.Current.FromCLI) == 0 {
			log.Fatal("error loading templates: ", err)
		}
	}

	initServices(c.DatabaseURL)

	// websockets
	hub := newHub(volatile)
	go hub.run()

	// Server Send Event, alternative to websocket
	b := realtime.NewBroker(func(ctx context.Context, key string) (string, error) {
		//TODO: Experimental, let un-authenticated user connect
		// useful for an Intercom-like SaaS I'm building.
		if strings.HasPrefix(key, "__tmp__experimental_public") {
			// let's create the most minimal authentication possible
			a := internal.Auth{
				AccountID: randStringRunes(30),
				UserID:    randStringRunes(30),
				Email:     "exp@tmp.com",
				Role:      0,
				Token:     key,
			}

			if err := volatile.SetTyped(key, a); err != nil {
				return key, err
			}

			return key, nil
		}

		auth, err := middleware.ValidateAuthKey(datastore, volatile, ctx, key)
		if err != nil {
			return "", err
		}

		// set base:token useful when executing pubsub event message / function
		conf, ok := ctx.Value(middleware.ContextBase).(internal.BaseConfig)
		if !ok {
			return "", errors.New("could not find base config")
		}

		//TODO: Lots of repetition of this, needs to be refactor
		if err := volatile.SetTyped(key, auth); err != nil {
			return "", err
		}
		if err := volatile.SetTyped("base:"+key, conf); err != nil {
			return "", err
		}

		return key, nil
	}, volatile)

	database := &Database{
		cache: volatile,
	}

	stdPub := []middleware.Middleware{
		middleware.Cors(),
	}

	pubWithDB := []middleware.Middleware{
		middleware.Cors(),
		middleware.WithDB(datastore, volatile),
	}

	stdAuth := []middleware.Middleware{
		middleware.Cors(),
		middleware.WithDB(datastore, volatile),
		middleware.RequireAuth(datastore, volatile),
	}

	stdRoot := []middleware.Middleware{
		middleware.WithDB(datastore, volatile),
		middleware.RequireRoot(datastore),
	}

	m := &membership{volatile: volatile}

	http.Handle("/login", middleware.Chain(http.HandlerFunc(m.login), pubWithDB...))
	http.Handle("/register", middleware.Chain(http.HandlerFunc(m.register), pubWithDB...))
	http.Handle("/email", middleware.Chain(http.HandlerFunc(m.emailExists), pubWithDB...))
	http.Handle("/password/resetcode", middleware.Chain(http.HandlerFunc(m.setResetCode), stdRoot...))
	http.Handle("/password/reset", middleware.Chain(http.HandlerFunc(m.resetPassword), pubWithDB...))
	//http.Handle("/setrole", chain(http.HandlerFunc(setRole), withDB))

	http.Handle("/sudogettoken/", middleware.Chain(http.HandlerFunc(m.sudoGetTokenFromAccountID), stdRoot...))

	// database routes
	http.Handle("/db/", middleware.Chain(http.HandlerFunc(database.dbreq), stdAuth...))
	http.Handle("/query/", middleware.Chain(http.HandlerFunc(database.query), stdAuth...))
	http.Handle("/inc/", middleware.Chain(http.HandlerFunc(database.increase), stdAuth...))
	http.Handle("/sudoquery/", middleware.Chain(http.HandlerFunc(database.query), stdRoot...))
	http.Handle("/sudolistall/", middleware.Chain(http.HandlerFunc(database.listCollections), stdRoot...))
	http.Handle("/sudo/index", middleware.Chain(http.HandlerFunc(database.index), stdRoot...))
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
	acct := &accounts{membership: m}
	http.Handle("/account/init", middleware.Chain(http.HandlerFunc(acct.create), stdPub...))
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

	// server-side functions
	f := &functions{datastore: datastore}
	http.Handle("/fn/add", middleware.Chain(http.HandlerFunc(f.add), stdRoot...))
	http.Handle("/fn/update", middleware.Chain(http.HandlerFunc(f.update), stdRoot...))
	http.Handle("/fn/delete/", middleware.Chain(http.HandlerFunc(f.del), stdRoot...))
	http.Handle("/fn/del/", middleware.Chain(http.HandlerFunc(f.del), stdRoot...))
	http.Handle("/fn/info/", middleware.Chain(http.HandlerFunc(f.info), stdRoot...))
	http.Handle("/fn/exec/", middleware.Chain(http.HandlerFunc(f.exec), stdAuth...))
	http.Handle("/fn", middleware.Chain(http.HandlerFunc(f.list), stdRoot...))

	// extras routes
	ex := &extras{}
	http.Handle("/extra/resizeimg", middleware.Chain(http.HandlerFunc(ex.resizeImage), stdAuth...))
	http.Handle("/extra/sms", middleware.Chain(http.HandlerFunc(ex.sudoSendSMS), stdRoot...))
	http.Handle("/extra/htmltox", middleware.Chain(http.HandlerFunc(ex.htmlToX), stdAuth...))

	// ui routes
	webUI := ui{}
	http.HandleFunc("/ui/login", webUI.auth)
	http.Handle("/ui/db", middleware.Chain(http.HandlerFunc(webUI.dbCols), stdRoot...))
	http.Handle("/ui/db/save", middleware.Chain(http.HandlerFunc(webUI.dbSave), stdRoot...))
	http.Handle("/ui/db/del/", middleware.Chain(http.HandlerFunc(webUI.dbDel), stdRoot...))
	http.Handle("/ui/db/", middleware.Chain(http.HandlerFunc(webUI.dbDoc), stdRoot...))
	http.Handle("/ui/fn/new", middleware.Chain(http.HandlerFunc(webUI.fnNew), stdRoot...))
	http.Handle("/ui/fn/save", middleware.Chain(http.HandlerFunc(webUI.fnSave), stdRoot...))
	http.Handle("/ui/fn/del/", middleware.Chain(http.HandlerFunc(webUI.fnDel), stdRoot...))
	http.Handle("/ui/fn/", middleware.Chain(http.HandlerFunc(webUI.fnEdit), stdRoot...))
	http.Handle("/ui/fn", middleware.Chain(http.HandlerFunc(webUI.fnList), stdRoot...))
	http.Handle("/ui/forms", middleware.Chain(http.HandlerFunc(webUI.forms), stdRoot...))
	http.Handle("/ui/forms/del/", middleware.Chain(http.HandlerFunc(webUI.formDel), stdRoot...))
	http.HandleFunc("/", webUI.login)

	// graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// handle stop/kill signal
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		<-c
		cancel()
	}()

	httpsvr := &http.Server{
		Addr: ":" + c.Port,
	}

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return httpsvr.ListenAndServe()
	})
	g.Go(func() error {
		<-gCtx.Done()
		return httpsvr.Shutdown(context.Background())
	})

	if err := g.Wait(); err != nil {
		fmt.Printf("exit reason: %s \n", err)
	}
}

func initServices(dbHost string) {

	if strings.EqualFold(dbHost, "mem") {
		volatile = cache.NewDevCache()
	} else {
		volatile = cache.NewCache()
	}

	persister := config.Current.DataStore
	if strings.EqualFold(dbHost, "mem") {
		datastore = memory.New(volatile.PublishDocument)
	} else if strings.EqualFold(persister, "mongo") {
		cl, err := openMongoDatabase(dbHost)
		if err != nil {
			log.Fatal(err)
		}
		datastore = mongo.New(cl, volatile.PublishDocument)
	} else {
		cl, err := openPGDatabase(dbHost)
		if err != nil {
			log.Fatal(err)
		}

		datastore = postgresql.New(cl, volatile.PublishDocument, "./sql/")
	}

	mp := config.Current.MailProvider
	if strings.EqualFold(mp, internal.MailProviderSES) {
		emailer = email.AWSSES{}
	} else {
		emailer = email.Dev{}
	}

	sp := config.Current.StorageProvider
	if strings.EqualFold(sp, internal.StorageProviderS3) {
		storer = storage.S3{}
	} else {
		storer = storage.Local{}
	}

	sub := &function.Subscriber{}
	sub.PubSub = volatile
	sub.GetExecEnv = func(token string) (function.ExecutionEnvironment, error) {
		var exe function.ExecutionEnvironment

		var conf internal.BaseConfig
		// for public websocket (experimental)
		if strings.HasPrefix(token, "__tmp__experimental_public") {
			pk := strings.Replace(token, "__tmp__experimental_public_", "", -1)
			pairs := strings.Split(pk, "_")
			fmt.Println("checking for base in cache: ", pairs[0])
			if err := volatile.GetTyped(pairs[0], &conf); err != nil {
				log.Println("cannot find base for public websocket")
				return exe, err
			}
		} else if err := volatile.GetTyped("base:"+token, &conf); err != nil {
			log.Println("cannot find base")
			return exe, err
		}

		var auth internal.Auth
		if err := volatile.GetTyped(token, &auth); err != nil {
			log.Println("cannot find auth")
			return exe, err
		}

		exe.Auth = auth
		exe.BaseName = conf.Name
		exe.DataStore = datastore
		exe.Volatile = volatile

		return exe, nil
	}

	// start system events subscriber
	go sub.Start()
}
func openMongoDatabase(dbHost string) (*mongodrv.Client, error) {
	uri := dbHost

	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	cl, err := mongodrv.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to mongo: %v", err)
	}

	if err := cl.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("Ping failed: %v", err)
	}

	return cl, nil
}

func openPGDatabase(dbHost string) (*sql.DB, error) {
	//connStr := "user=postgres password=example dbname=test sslmode=disable"
	dbConn, err := sql.Open("postgres", dbHost)
	if err != nil {
		return nil, err
	}

	if err := dbConn.Ping(); err != nil {
		return nil, err
	}

	return dbConn, nil
}

func ping(w http.ResponseWriter, r *http.Request) {
	if err := datastore.Ping(); err != nil {
		http.Error(w, "connection failed to database, I'm down.", http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusOK, true)
}

func sudoCache(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodGet {
		typ := r.URL.Query().Get("type")
		key := fmt.Sprintf("%s_%s", conf.Name, r.URL.Query().Get("key"))

		if typ == "queue" {
			val, err := volatile.DequeueWork(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			respond(w, http.StatusOK, val)
			return
		}
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
			Type  string `json:"type"`
		})
		if err := parseBody(r.Body, &data); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		data.Key = fmt.Sprintf("%s_%s", conf.Name, data.Key)

		if data.Type == "queue" {
			if err := volatile.QueueWork(data.Key, data.Value); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			respond(w, http.StatusOK, true)
			return
		}
		if err := volatile.Set(data.Key, data.Value); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		respond(w, http.StatusOK, true)
	}
}

func getURLPart(s string, idx int) string {
	parts := strings.Split(s, "/")
	if len(parts) <= idx {
		return ""
	}
	return parts[idx]
}
