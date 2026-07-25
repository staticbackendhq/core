// Package backend allows a Go program to import a standard Go package
// instead of self-hosting the backend API in a separate web server.
//
// You need to call the [Setup] function to initialize all services passing
// a [github.com/staticbackendhq/core/config.AppConfig]. You may create
// environment variables and load the config directly by confing.Load function.
//
//	// You may load the configuration from environment variables
//	// config.Current = config.LoadConfig()
//
//	// this sample uses the in-memory database provider built-in
//	// you can use PostgreSQL or MongoDB
//	config.Current = config.AppConfig{
//	  AppEnv:      "dev",
//	  DataStore:   "mem",
//	  DatabaseURL: "mem",
//	  LocalStorageURL: "http://localhost:8099",
//	}
//	backend.Setup(config.Current)
//
// The building blocks of [StaticBackend] are exported as variables and can be
// used directly accessing their interface's functions. For instance
// to use the [github.com/staticbackendhq/core/cache.Volatilizer] (cache and
// pub/sub) you'd use the [Cache] variable:
//
//	if err := backend.Cache.Set("key", "value"); err != nil {
//	  return err
//	}
//	val, err := backend.Cache.Get("key")
//
// The available services are as follow:
//   - [Cache]: caching and pub/sub
//   - [DB]: a raw [github.com/staticbackendhq/core/database.Persister] instance (see below for when to use it)
//   - [Filestore]: raw blob storage
//   - [Emailer]: to send emails
//   - [Config]: the config that was passed to [Setup]
//
// You may see those services as raw building blocks that give you the most
// flexibility to build on top.
//
// For easy of use, this package wraps important / commonly used
// functionalities into more developer friendly implementations.
//
// For instance, the [Membership] function wants a
// [github.com/staticbackendhq/core/model.DatabaseConfig] and allows the caller
// to create account and user as well as reseting password etc.
//
//	usr := backend.Membership(base)
//	sessionToken, user, err := usr.CreateAccountAndUser("me@test.com", "passwd", 100)
//
// To contrast, all of those can be done from your program by using the [DB]
// ([github.com/staticbackendhq/core/database.Persister]) data store, but for
// convenience this package offers easier / ready-made functions for common
// use-cases. Example for database CRUD and querying:
//
//	tasks := backend.Collection[Task](auth, base, "tasks")
//	newTask, err := tasks.Create(Task{Name: "testing"})
//
// The [Collection] returns a strongly-typed structure where functions
// input/output are properly typed, it's a generic type.
//
// [StaticBackend] makes your Go web application multi-tenant by default.
// For this reason you must supply a
// [github.com/staticbackendhq/core/model.DatabaseConfig] and (database) and
// sometimes a [github.com/staticbackendhq/core/model.Auth] (user performing
// the actions) to the different parts of the system so the data and security
// are applied to the right tenant, account and user.
//
// You'd design your application around one or more tenants. Each tenant has
// their own database. It's fine to have one tenant/database. In that case
// you might create the tenant and its database and use the database ID in
// an environment variable. From a middleware you might load the database from
// this ID.
//
//	// if you'd want to use SB's middleware (it's not required)
//	// you use whatever you like for your web handlers and middleware.
//	// SB is a library not a framework.
//	func DetectTenant() middleware.Middleware {
//	  return func(next http.Handler) http.Handler {
//	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	      // check for presence of a public DB ID
//	      // this can come from cookie, URL query param
//	      key := r.Header.Get("DB-ID")
//	      // for multi-tenant, DB ID can be from an env var
//	      if len(key) == 0 {
//	        key = os.Getenv("SINGLE_TENANT_DBID")
//	      }
//	      var curDB model.DatabaseConfig
//	      if err := backend.Cache.GetTyped(key, &curDB); err != nil {
//	        http.Error(w, err.Error(), http.StatusBadRequest)
//	        return
//	      }
//	      curDB, err := backend.DB.FindDatabase(key)
//	      // err != nil return HTTP 400 Bad request
//	      err = backend.Cache.SetTyped(key, curDB)
//	      // add the tenant's DB in context for the rest of
//	      // your pipeline to have the proper DB.
//	      ctx := r.Context()
//	      ctx = context.WithValue(ctx, ContextBase, curDB)
//	      next.ServeHTTP(w, r.WithContext(ctx)))
//	    })
//	  }
//	}
//
// You'd create a similar middleware for adding the current user into the
// request context.
//
// If you ever decide to switch to a multi-tenant design, you'd already be all
// set with this middleware, instead of getting the ID from the env variable,
// you'd define how the user should provide their database ID.
//
// [StaticBackend]: https://staticbackend.dev/
package backend

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/database"
	"github.com/staticbackendhq/core/database/memory"
	"github.com/staticbackendhq/core/database/mongo"
	"github.com/staticbackendhq/core/database/postgresql"
	"github.com/staticbackendhq/core/database/sqlite"
	"github.com/staticbackendhq/core/email"
	"github.com/staticbackendhq/core/function"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/model"
	"github.com/staticbackendhq/core/search"
	"github.com/staticbackendhq/core/storage"
	mongodrv "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

// All StaticBackend services (need to call Setup before using them).
var (
	// Config reflect the configuration received on Setup
	Config config.AppConfig

	// DB initialized Persister data store
	DB database.Persister
	// Emailer initialized Mailer for sending emails
	Emailer email.Mailer
	// Filestore initialized Storer for raw save/delete blob file
	Filestore storage.Storer
	// Cache initialized Volatilizer for cache and pub/sub
	Cache  cache.Volatilizer
	Search *search.Search

	// Membership exposes Account and User functionalities like register, login, etc
	// account and user functionalities.
	Membership func(model.DatabaseConfig) User

	// Storage exposes file storage functionalities. It wraps the blob
	// storage as well as the database storage.
	Storage func(model.Auth, model.DatabaseConfig) FileStore

	// Scheduler to execute schedule jobs (only on PrimaryInstance)
	Scheduler *function.TaskScheduler

	lifecycleMu      sync.Mutex
	closeOnce        sync.Once
	closeErr         error
	subscriberCancel context.CancelFunc
	subscriberDone   chan struct{}
)

// Setup initializes the core services based on the configuration received.
func Setup(cfg config.AppConfig) {
	logger.Setup(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := Close(ctx); err != nil {
		slog.Error("error closing existing backend services", "error", err)
	}

	Config = cfg
	resetLifecycle()

	if strings.EqualFold(cfg.DatabaseURL, "mem") || strings.EqualFold(cfg.RedisHost, "mem") {
		Cache = cache.NewDevCache()
	} else {
		Cache = cache.NewCache()
	}

	persister := config.Current.DataStore
	if strings.EqualFold(cfg.DatabaseURL, "mem") {
		DB = memory.New(Cache.PublishDocument)
	} else if strings.EqualFold(persister, "mongo") {
		cl, err := openMongoDatabase(cfg.DatabaseURL)
		if err != nil {
			logger.FatalError("failed to create connection with mongodb", err)
		}
		DB = mongo.New(cl, Cache.PublishDocument)
	} else if strings.EqualFold(persister, "sqlite") {
		cl, err := openSQLite(cfg.DatabaseURL)
		if err != nil {
			logger.FatalError("failed to create connection with SQLite", err)
		}

		DB = sqlite.New(cl, Cache.PublishDocument)
	} else {
		cl, err := openPGDatabase(cfg.DatabaseURL, cfg)
		if err != nil {
			logger.FatalError("failed to create connection with postgres", err)
		}
		pool := normalizedPostgresPoolConfig(cfg)
		Log.Info().
			Int("max_open_connections", pool.maxOpenConns).
			Int("max_idle_connections", pool.maxIdleConns).
			Int("max_lifetime_seconds", pool.maxLifetimeSeconds).
			Int("max_idle_time_seconds", pool.maxIdleTimeSeconds).
			Msg("postgres connection pool configured")

		DB = postgresql.New(cl, Cache.PublishDocument)
	}

	mp := cfg.MailProvider
	if strings.EqualFold(mp, email.MailProviderSES) {
		Emailer = email.AWSSES{}
	} else if strings.EqualFold(mp, email.MailProviderLocal) {
		Emailer = email.Local{}
	} else {
		Emailer = email.Dev{}
	}

	sp := cfg.StorageProvider
	if strings.EqualFold(sp, storage.StorageProviderS3) {
		Filestore = storage.S3{}
	} else {
		Filestore = storage.Local{}
	}

	if !cfg.NoFullTextSearch {
		ftsFilename := cfg.FullTextIndexFile
		if len(ftsFilename) == 0 {
			ftsFilename = "sb.fts"
		}
		src, err := search.New(ftsFilename, Cache)
		if err != nil {
			logger.FatalError("unable to start full-text search", err)
			return
		}

		Search = src
	}

	sub := &function.Subscriber{}
	sub.PubSub = Cache
	sub.GetExecEnv = func(msg model.Command) (*function.ExecutionEnvironment, error) {
		exe := &function.ExecutionEnvironment{
			Auth:      msg.Auth,
			BaseName:  msg.Base,
			DataStore: DB,
			Volatile:  Cache,
			Search:    Search,
			Email:     Emailer,
		}

		return exe, nil
	}

	isPrimary := false
	if len(cfg.PrimaryInstanceHostname) == 0 {
		// if no value is provided, like on GH action for tests, we assume primary
		isPrimary = true
	} else if hostname, err := os.Hostname(); err != nil {
		slog.Warn("cannot determine if it's primary instance", "error", err)
	} else if strings.EqualFold(hostname, cfg.PrimaryInstanceHostname) {
		isPrimary = true
	}

	sub.IsPrimaryInstance = isPrimary

	// start system events subscriber
	subCtx, cancel := context.WithCancel(context.Background())
	lifecycleMu.Lock()
	subscriberCancel = cancel
	subscriberDone = make(chan struct{})
	done := subscriberDone
	lifecycleMu.Unlock()
	go func() {
		defer close(done)
		sub.StartContext(subCtx)
	}()

	// for primary instance, we start the job scheduler
	if isPrimary {
		runner := &function.TaskScheduler{
			Volatile:  Cache,
			DataStore: DB,
			Search:    Search,
			Email:     Emailer,
		}

		Scheduler = runner
		go runner.Start()
		slog.Info("job scheduler / runner started on primary instance")
	}

	Membership = newUser
	Storage = newFile
}

// Close gracefully stops services started by Setup.
func Close(ctx context.Context) error {
	closeOnce.Do(func() {
		closeErr = closeServices(ctx)
	})

	return closeErr
}

func resetLifecycle() {
	lifecycleMu.Lock()
	defer lifecycleMu.Unlock()

	closeOnce = sync.Once{}
	closeErr = nil
	subscriberCancel = nil
	subscriberDone = nil
}

func closeServices(ctx context.Context) error {
	var errs []error

	if Scheduler != nil {
		if err := Scheduler.Stop(ctx); err != nil {
			errs = append(errs, err)
		}
		Scheduler = nil
	}

	lifecycleMu.Lock()
	cancel := subscriberCancel
	done := subscriberDone
	lifecycleMu.Unlock()
	if cancel != nil {
		cancel()
		select {
		case <-done:
		case <-ctx.Done():
			errs = append(errs, ctx.Err())
		}
	}

	if Search != nil {
		Search.Close()
		Search = nil
	}

	if err := closeResource(ctx, DB); err != nil {
		errs = append(errs, err)
	}
	if err := closeResource(ctx, Cache); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func closeResource(ctx context.Context, resource any) error {
	switch closer := resource.(type) {
	case interface{ Close(context.Context) error }:
		return closer.Close(ctx)
	case interface{ Close() error }:
		return closer.Close()
	default:
		return nil
	}
}

func openMongoDatabase(dbHost string) (*mongodrv.Client, error) {
	uri := dbHost

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cl, err := mongodrv.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to mongo: %v", err)
	}

	if err := cl.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("ping failed: %v", err)
	}

	return cl, nil
}

const (
	defaultPostgresMaxOpenConns           = 10
	defaultPostgresMaxIdleConns           = 5
	defaultPostgresConnMaxLifetimeSeconds = 1800
	defaultPostgresConnMaxIdleTimeSeconds = 300
)

type postgresPoolConfig struct {
	maxOpenConns       int
	maxIdleConns       int
	maxLifetimeSeconds int
	maxIdleTimeSeconds int
}

func normalizedPostgresPoolConfig(cfg config.AppConfig) postgresPoolConfig {
	pool := postgresPoolConfig{
		maxOpenConns:       cfg.PostgresMaxOpenConns,
		maxIdleConns:       cfg.PostgresMaxIdleConns,
		maxLifetimeSeconds: cfg.PostgresConnMaxLifetimeSeconds,
		maxIdleTimeSeconds: cfg.PostgresConnMaxIdleTimeSeconds,
	}

	if pool.maxOpenConns <= 0 {
		pool.maxOpenConns = defaultPostgresMaxOpenConns
	}
	if pool.maxIdleConns <= 0 {
		pool.maxIdleConns = defaultPostgresMaxIdleConns
	}
	if pool.maxIdleConns > pool.maxOpenConns {
		pool.maxIdleConns = pool.maxOpenConns
	}
	if pool.maxLifetimeSeconds <= 0 {
		pool.maxLifetimeSeconds = defaultPostgresConnMaxLifetimeSeconds
	}
	if pool.maxIdleTimeSeconds <= 0 {
		pool.maxIdleTimeSeconds = defaultPostgresConnMaxIdleTimeSeconds
	}

	return pool
}

func configurePostgresPool(db *sql.DB, cfg config.AppConfig) postgresPoolConfig {
	pool := normalizedPostgresPoolConfig(cfg)
	db.SetMaxOpenConns(pool.maxOpenConns)
	db.SetMaxIdleConns(pool.maxIdleConns)
	db.SetConnMaxLifetime(time.Duration(pool.maxLifetimeSeconds) * time.Second)
	db.SetConnMaxIdleTime(time.Duration(pool.maxIdleTimeSeconds) * time.Second)

	return pool
}

func openPGDatabase(dbHost string, cfg config.AppConfig) (*sql.DB, error) {
	//connStr := "user=postgres password=example dbname=test sslmode=disable"
	dbConn, err := sql.Open("postgres", dbHost)
	if err != nil {
		return nil, err
	}
	configurePostgresPool(dbConn, cfg)

	if err := dbConn.Ping(); err != nil {
		_ = dbConn.Close()
		return nil, err
	}

	return dbConn, nil
}

func openSQLite(url string) (*sql.DB, error) {
	dbConn, err := sql.Open("sqlite", url)
	if err != nil {
		return nil, err
	}

	if err := dbConn.Ping(); err != nil {
		return nil, err
	}

	return dbConn, nil
}
