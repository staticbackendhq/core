package backend

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/database/memory"
	"github.com/staticbackendhq/core/database/mongo"
	"github.com/staticbackendhq/core/database/postgresql"
	"github.com/staticbackendhq/core/email"
	"github.com/staticbackendhq/core/function"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/middleware"
	"github.com/staticbackendhq/core/storage"
	mongodrv "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	_ "github.com/lib/pq"
)

type Backend struct {
	Tenant Tenant
	//DB     func(token, baseID string) Database
	User func(baseID string) User
}

var (
	datastore internal.Persister
	emailer   internal.Mailer
	filestore internal.Storer

	// Cache exposes the cache / pub-sub functionalities
	Cache internal.Volatilizer
	// Log expose the configured logger
	Log *logger.Logger
)

// Init prepares the backend core based on the configuration
func New(cfg config.AppConfig) Backend {
	//TODO: this code is an awfuly copy of the server.go init code
	// Might be an idea to create some kind of helper in the internal package
	// that would return a structure with all the initialized services

	Log = logger.Get(cfg)

	if strings.EqualFold(cfg.DatabaseURL, "mem") {
		Cache = cache.NewDevCache(Log)
	} else {
		Cache = cache.NewCache(Log)
	}

	persister := config.Current.DataStore
	if strings.EqualFold(cfg.DatabaseURL, "mem") {
		datastore = memory.New(Cache.PublishDocument)
	} else if strings.EqualFold(persister, "mongo") {
		cl, err := openMongoDatabase(cfg.DatabaseURL)
		if err != nil {
			Log.Fatal().Err(err).Msg("failed to create connection with mongodb")
		}
		datastore = mongo.New(cl, Cache.PublishDocument, Log)
	} else {
		cl, err := openPGDatabase(cfg.DatabaseURL)
		if err != nil {
			Log.Fatal().Err(err).Msg("failed to create connection with postgres")
		}

		datastore = postgresql.New(cl, Cache.PublishDocument, "./sql/", Log)
	}

	mp := cfg.MailProvider
	if strings.EqualFold(mp, internal.MailProviderSES) {
		emailer = email.AWSSES{}
	} else {
		emailer = email.Dev{}
	}

	sp := cfg.StorageProvider
	if strings.EqualFold(sp, internal.StorageProviderS3) {
		filestore = storage.S3{}
	} else {
		filestore = storage.Local{}
	}

	sub := &function.Subscriber{Log: Log}
	sub.PubSub = Cache
	sub.GetExecEnv = func(token string) (function.ExecutionEnvironment, error) {
		var exe function.ExecutionEnvironment

		var conf internal.BaseConfig
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

		var auth internal.Auth
		if err := Cache.GetTyped(token, &auth); err != nil {
			Log.Error().Err(err).Msg("cannot find auth")
			return exe, err
		}

		exe.Auth = auth
		exe.BaseName = conf.Name
		exe.DataStore = datastore
		exe.Volatile = Cache

		return exe, nil
	}

	// start system events subscriber
	go sub.Start()

	return Backend{
		Tenant: Tenant{},
		//DB:     newDatabase,
		User: newUser,
	}
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

func NewID() string {
	return datastore.NewID()
}

func findAuth(token string) internal.Auth {
	auth, err := middleware.ValidateAuthKey(datastore, Cache, context.Background(), token)
	if err != nil {
		return internal.Auth{}
	}
	return auth
}

func findBase(baseID string) internal.BaseConfig {
	var conf internal.BaseConfig
	if err := Cache.GetTyped(baseID, &conf); err != nil {
		db, err := datastore.FindDatabase(baseID)
		if err != nil {
			return conf
		}
		conf = db
	}
	return conf
}
