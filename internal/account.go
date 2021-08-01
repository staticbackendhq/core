package internal

import (
	"context"
	"time"

	"github.com/gbrlsnchs/jwt/v3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// Auth represents an authenticated user.
type Auth struct {
	AccountID primitive.ObjectID
	UserID    primitive.ObjectID
	Email     string
	Role      int
}

// JWTPayload contains the current user token
type JWTPayload struct {
	jwt.Payload
	Token string `json:"token,omitempty"`
}

var (
	ctx = context.Background()
)

type Account struct {
	ID    primitive.ObjectID `bson:"_id" json:"id"`
	Email string             `bson:"email" json:"email"`
}

type Token struct {
	ID        primitive.ObjectID `bson:"_id" json:"id"`
	AccountID primitive.ObjectID `bson:"accountId" json:"accountId"`
	Token     string             `bson:"token" json:"token"`
	Email     string             `bson:"email" json:"email"`
	Password  string             `bson:"pw" json:"-"`
	Role      int                `bson:"role" json:"role"`
}

type Login struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Customer struct {
	ID               primitive.ObjectID `bson:"_id" json:"id"`
	Email            string             `bson:"email" json:"email"`
	StripeID         string             `bson:"stripeId" json:"stripeId"`
	SubscriptionID   string             `bson:"subId" json:"subId"`
	IsActive         bool               `bson:"active" json:"-"`
	MonthlyEmailSent int                `bson:"mes" json:"-"`
	Created          time.Time          `bson:"created" json:"created"`
}

func EmailExists(db *mongo.Database, email string) (bool, error) {
	count, err := db.Collection("accounts").CountDocuments(ctx, bson.M{"email": email})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func FindToken(db *mongo.Database, id primitive.ObjectID, token string) (tok Token, err error) {
	sr := db.Collection("sb_tokens").FindOne(ctx, bson.M{FieldID: id, FieldToken: token})
	err = sr.Decode(&tok)
	return
}

func FindRootToken(db *mongo.Database, id, accountID primitive.ObjectID, token string) (tok Token, err error) {
	filter := bson.M{
		FieldID:        id,
		FieldAccountID: accountID,
		FieldToken:     token,
	}
	sr := db.Collection("sb_tokens").FindOne(ctx, filter)
	err = sr.Decode(&tok)
	return
}

func FindTokenByEmail(db *mongo.Database, email string) (tok Token, err error) {
	sr := db.Collection("sb_tokens").FindOne(ctx, bson.M{"email": email})
	err = sr.Decode(&tok)
	return
}

func FindAccount(db *mongo.Database, accountID primitive.ObjectID) (cus Customer, err error) {
	filter := bson.M{FieldID: accountID}
	sr := db.Collection("accounts").FindOne(ctx, filter)
	err = sr.Decode(&cus)
	return
}

func CreateAccount(db *mongo.Database, cus Customer) error {
	if _, err := db.Collection("accounts").InsertOne(ctx, cus); err != nil {
		return err
	}
	return nil
}

func CreateBase(db *mongo.Database, base BaseConfig) error {
	if _, err := db.Collection("bases").InsertOne(ctx, base); err != nil {
		return err
	}
	return nil
}

func FindDatabase(db *mongo.Database, id primitive.ObjectID) (conf BaseConfig, err error) {
	sr := db.Collection("bases").FindOne(ctx, bson.M{FieldID: id})
	err = sr.Decode(&conf)
	return
}

func DatabaseExists(db *mongo.Database, name string) (bool, error) {
	count, err := db.Collection("bases").CountDocuments(ctx, bson.M{"name": name})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
