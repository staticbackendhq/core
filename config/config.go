package config

import "os"

var Current AppConfig

type AppConfig struct {
	// Port web server port
	Port string

	// AppEnv represent the environment in which the server runs
	AppEnv string
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

	// AWSRegion region for AWS
	AWSRegion string
	// AWSS3Bucket S3 bucket
	AWSS3Bucket string
	// AWSCDNURL CDN URL
	AWSCDNURL string

	// KeepPermissionInName if "yes" will keep the repo permission in repo name
	KeepPermissionInName string
}

func LoadConfig() AppConfig {
	return AppConfig{
		Port:                  os.Getenv("PORT"),
		AppEnv:                os.Getenv("APP_ENV"),
		FromCLI:               os.Getenv("SB_FROM_CLI"),
		DataStore:             os.Getenv("DATA_STORE"),
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		MailProvider:          os.Getenv("MAIL_PROVIDER"),
		FromEmail:             os.Getenv("FROM_EMAIL"),
		FromName:              os.Getenv("FROM_NAME"),
		StorageProvider:       os.Getenv("STORAGE_PROVIDER"),
		LocalStorageURL:       os.Getenv("LOCAL_STORAGE_URL"),
		RedisURL:              os.Getenv("REDIS_URL"),
		RedisHost:             os.Getenv("REDIS_HOST"),
		RedisPassword:         os.Getenv("REDIS_PASSWORD"),
		StripeKey:             os.Getenv("STRIPE_KEY"),
		StripePriceIDIdea:     os.Getenv("STRIPE_PRICEID_IDEA"),
		StripePriceIDLaunch:   os.Getenv("STRIPE_PRICEID_LAUNCH"),
		StripePriceIDTraction: os.Getenv("STRIPE_PRICEID_TRACTION"),
		StripePriceIDGrowth:   os.Getenv("STRIPE_PRICEID_GROWTH"),
		StripeWebhookSecret:   os.Getenv("STRIPE_WEBHOOK_SECRET"),
		TwilioAccountID:       os.Getenv("TWILIO_ACCOUNTSID"),
		TwilioAuthToken:       os.Getenv("TWILIO_AUTHTOKEN"),
		TwilioTestCellNumber:  os.Getenv("MY_CELL"),
		TwilioNumber:          os.Getenv("TWILIO_NUMBER"),
		AWSRegion:             os.Getenv("AWS_REGION"),
		AWSCDNURL:             os.Getenv("AWS_CDN_URL"),
		AWSS3Bucket:           os.Getenv("AWS_S3_BUCKET"),
		KeepPermissionInName:  os.Getenv("KEEP_PERM_COL_NAME"),
	}
}
