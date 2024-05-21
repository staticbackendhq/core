package config

import "os"

var Current AppConfig

type AppConfig struct {
	// PrimaryInstanceHostname indicates the name of the instance that should run
	// the scheduler. Only one instance can run the scheduler.
	PrimaryInstanceHostname string

	// Port web server port
	Port string

	// AppEnv represent the environment in which the server runs
	AppEnv string
	// AppSecre used for encryption/decryption
	AppSecret string
	// AppURL is the full URL of the backend (important for social logins callbacks)
	AppURL string
	// FromCLI if we're running in the CLI
	FromCLI string

	// DataStore used as the data store implementation
	DataStore string
	// DatabaseURL is the database URL
	DatabaseURL string

	// StorageProvider used as the file storage implementation
	StorageProvider string
	// LocalStorageURLURL for files when using local storage provider
	LocalStorageURL string

	// MailProvider used as the sending mails implementeation
	MailProvider string
	// FromEmail used when SB sends email
	FromEmail string
	// FromName used when SB sends email
	FromName string

	// StripeKey used for Stripe communication
	StripeKey string
	// StripePriceIDIdea is the price id for default Stripe plan
	StripePriceIDIdea string
	// StripePriceIDLaunch is the price id for Launch plan
	StripePriceIDLaunch string
	// StripePriceIDTraction is the price id for Traction plan
	StripePriceIDTraction string
	// StripePriceIDGrowth is the price id for Growth plan
	StripePriceIDGrowth string
	// StripeWebhookSecret used when Stripe sends a webhook
	StripeWebhookSecret string

	// TwilioAccountID used when sending SMS text messages via Twilio API
	TwilioAccountID string
	// TwilioAuthToken used when sending SMS text messages via Twilio API
	TwilioAuthToken string
	// TwilioTestCellNumber used to perform unit test of Twilio API
	TwilioTestCellNumber string
	// TwilioNumber is the Twilio phone number used to send SMS text messages
	TwilioNumber string

	// RedisURL URL for Redis
	RedisURL string
	// RedisHost if RedisURL is not used, host for Redis
	RedisHost string
	// RedisPassword if RedisURL is not used, password for Redis
	RedisPassword string

	// S3AccessKey access key for S3 connection
	S3AccessKey string
	// S3SecretKey secret key for S3 connection
	S3SecretKey string
	// S3Endpoint endpoint for the S3 compat API
	S3Endpoint string
	// S3Region region for AWS
	S3Region string
	// S3Bucket S3 bucket
	S3Bucket string
	// S3CDNURL CDN URL
	S3CDNURL string

	// KeepPermissionInName if "yes" will keep the repo permission in repo name
	KeepPermissionInName bool

	// LogConsoleLevel could be use to specify the minimum log level is wanted
	LogConsoleLevel string

	// LogFilename if set, write logs to console and this file.
	LogFilename string
	// NoFullTextSearch prevents full-text search index from initializing
	NoFullTextSearch bool
	// FullTextIndexFile fully qualify file path for the search index
	// Hint: this is usually on a disk that do not vanish on each deployment.
	FullTextIndexFile string
	// ActivateFlag when set, the /account/init can bypass Stripe if matching val
	ActivateFlag string
	// PluginsPath is the full qualified path where plugins are stored
	PluginsPath string
}

func LoadConfig() AppConfig {
	return AppConfig{
		PrimaryInstanceHostname: os.Getenv("PRIMARY_INSTANCE_HOSTNAME"),
		Port:                    os.Getenv("PORT"),
		AppEnv:                  os.Getenv("APP_ENV"),
		AppSecret:               os.Getenv("APP_SECRET"),
		AppURL:                  os.Getenv("APP_URL"),
		FromCLI:                 os.Getenv("SB_FROM_CLI"),
		DataStore:               os.Getenv("DATA_STORE"),
		DatabaseURL:             os.Getenv("DATABASE_URL"),
		MailProvider:            os.Getenv("MAIL_PROVIDER"),
		FromEmail:               os.Getenv("FROM_EMAIL"),
		FromName:                os.Getenv("FROM_NAME"),
		StorageProvider:         os.Getenv("STORAGE_PROVIDER"),
		LocalStorageURL:         os.Getenv("LOCAL_STORAGE_URL"),
		RedisURL:                os.Getenv("REDIS_URL"),
		RedisHost:               os.Getenv("REDIS_HOST"),
		RedisPassword:           os.Getenv("REDIS_PASSWORD"),
		StripeKey:               os.Getenv("STRIPE_KEY"),
		StripePriceIDIdea:       os.Getenv("STRIPE_PRICEID_IDEA"),
		StripePriceIDLaunch:     os.Getenv("STRIPE_PRICEID_LAUNCH"),
		StripePriceIDTraction:   os.Getenv("STRIPE_PRICEID_TRACTION"),
		StripePriceIDGrowth:     os.Getenv("STRIPE_PRICEID_GROWTH"),
		StripeWebhookSecret:     os.Getenv("STRIPE_WEBHOOK_SECRET"),
		TwilioAccountID:         os.Getenv("TWILIO_ACCOUNTSID"),
		TwilioAuthToken:         os.Getenv("TWILIO_AUTHTOKEN"),
		TwilioTestCellNumber:    os.Getenv("MY_CELL"),
		TwilioNumber:            os.Getenv("TWILIO_NUMBER"),
		S3AccessKey:             os.Getenv("S3_ACCESSKEY"),
		S3SecretKey:             os.Getenv("S3_SECRETKEY"),
		S3Endpoint:              os.Getenv("S3_ENDPOINT"),
		S3Region:                os.Getenv("S3_REGION"),
		S3Bucket:                os.Getenv("S3_BUCKET"),
		S3CDNURL:                os.Getenv("S3_CDN_URL"),
		KeepPermissionInName:    os.Getenv("KEEP_PERM_COL_NAME") == "",
		LogConsoleLevel:         os.Getenv("LOG_CONSOLE_LEVEL"),
		LogFilename:             os.Getenv("LOG_FILENAME"),
		FullTextIndexFile:       os.Getenv("FTS_INDEX_FILE"),
		ActivateFlag:            os.Getenv("ACTIVATE_FLAG"),
		PluginsPath:             os.Getenv("PLUGINS_PATH"),
	}
}
