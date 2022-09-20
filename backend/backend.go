// Package backend allows a Go program to import a standard Go package
// instead of self-hosting the backend API.
//
// You need to call the Setup function to initialize all services passing
// a config.AppConfig. You may create environment variables and load the config
// directly from confing.LoadConfig.
//
// The building blocks of StaticBackend are exported as variable and can be
// used directly accessing their interface's functions. For instance
// to use the Volatilizer (cache and pub/sub) you'd use the Cache variable:
//
//    if err := backend.Cache.Set("key", "value"); err != nil {
//      return err
//    }
//    val, err := backend.Cache.Get("key")
//
// The available services are as follow:
// 1. Cache: caching and pub/sub
// 2. DB: a raw Persister instance (see below for when to use it)
// 3. Filestore: raw blob storage
// 4. Emailer: to send emails
// 5. Config: the config that was passed to Setup
// 6. Log: logger
//
// You may see those services as raw building blocks to give you the most
// flexibility. For easy of use, this package wraps important / commonly used
// functionalities into more developer friendly implementation.
//
// For instance, the Membership function wants a model.DatabaseConfig and allow
// the caller to create account and user as well as reseting password etc.
//
// To contrast, all of those can be done from your program by using the DB
// (Persister) data store, but for convenience this package offers easier /
// ready-made functions for common use-case.
//
// StaticBackend makes your Go web application multi-tenant by default.
// For this reason you must supply a model.DatabaseConfig and sometimes a
// model.auth (user performing the actions) to the different parts of the system
// so the data and security are applied to the right tenant, account and user.
package backend

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/database"
	"github.com/staticbackendhq/core/database/memory"
	"github.com/staticbackendhq/core/database/mongo"
	"github.com/staticbackendhq/core/database/postgresql"
	"github.com/staticbackendhq/core/email"
	"github.com/staticbackendhq/core/function"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/middleware"
	"github.com/staticbackendhq/core/model"
	"github.com/staticbackendhq/core/storage"
	mongodrv "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	_ "github.com/lib/pq"
)

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
	Cache cache.Volatilizer
	// Log initialized Logger for all logging
	Log *logger.Logger

	// Membership exposes Account and User functionalities like register, login, etc
	// account and user functionalities.
	Membership func(model.DatabaseConfig) User

	// Storage exposes file storage functionalities. It wraps the blob
	// storage as well as the database storage.
	Storage func(model.Auth, model.DatabaseConfig) FileStore
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Setup initializes the core services based on the configuration received.
func Setup(cfg config.AppConfig) {
	Config = cfg

	Log = logger.Get(cfg)

	if strings.EqualFold(cfg.DatabaseURL, "mem") {
		Cache = cache.NewDevCache(Log)
	} else {
		Cache = cache.NewCache(Log)
	}

	persister := config.Current.DataStore
	if strings.EqualFold(cfg.DatabaseURL, "mem") {
		DB = memory.New(Cache.PublishDocument)
	} else if strings.EqualFold(persister, "mongo") {
		cl, err := openMongoDatabase(cfg.DatabaseURL)
		if err != nil {
			Log.Fatal().Err(err).Msg("failed to create connection with mongodb")
		}
		DB = mongo.New(cl, Cache.PublishDocument, Log)
	} else {
		cl, err := openPGDatabase(cfg.DatabaseURL)
		if err != nil {
			Log.Fatal().Err(err).Msg("failed to create connection with postgres")
		}

		DB = postgresql.New(cl, Cache.PublishDocument, "./sql/", Log)
	}

	mp := cfg.MailProvider
	if strings.EqualFold(mp, email.MailProviderSES) {
		Emailer = email.AWSSES{}
	} else {
		Emailer = email.Dev{}
	}

	sp := cfg.StorageProvider
	if strings.EqualFold(sp, storage.StorageProviderS3) {
		Filestore = storage.S3{}
	} else {
		Filestore = storage.Local{}
	}

	sub := &function.Subscriber{Log: Log}
	sub.PubSub = Cache
	sub.GetExecEnv = func(token string) (function.ExecutionEnvironment, error) {
		var exe function.ExecutionEnvironment

		var conf model.DatabaseConfig
		// for public websocket (experimental)
		if strings.HasPrefix(token, "__tmp__experimental_public") {
			pk := strings.Replace(token, "__tmp__experimental_public_", "", -1)
			pairs := strings.Split(pk, "_")
			Log.Info().Msgf("checking for base in cache: %s", pairs[0])
			if err := Cache.GetTyped(pairs[0], &conf); err != nil {
				Log.Error().Err(err).Msg("cannot find base for public websocket")
				return exe, err
			}
		} else if err := Cache.GetTyped("base:"+token, &conf); err != nil {
			Log.Error().Err(err).Msg("cannot find base")
			return exe, err
		}

		var auth model.Auth
		if err := Cache.GetTyped(token, &auth); err != nil {
			Log.Error().Err(err).Msg("cannot find auth")
			return exe, err
		}

		exe.Auth = auth
		exe.BaseName = conf.Name
		exe.DataStore = DB
		exe.Volatile = Cache

		return exe, nil
	}

	// start system events subscriber
	go sub.Start()

	Membership = newUser
	Storage = newFile
}

func openMongoDatabase(dbHost string) (*mongodrv.Client, error) {
	uri := dbHost

	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	cl, err := mongodrv.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("cannot connect to mongo: %v", err)
	}

	if err := cl.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("ping failed: %v", err)
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

func findAuthz(token string) model.Auth {
	auth, err := middleware.ValidateAuthKey(DB, Cache, context.Background(), token)
	if err != nil {
		return model.Auth{}
	}
	return auth
}

func findBasez(baseID string) model.DatabaseConfig {
	var conf model.DatabaseConfig
	if err := Cache.GetTyped(baseID, &conf); err != nil {
		db, err := DB.FindDatabase(baseID)
		if err != nil {
			return conf
		}
		conf = db
	}
	return conf
}
