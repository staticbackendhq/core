package internal

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ctx = context.Background()
)

const (
	FieldID        = "_id"
	FieldAccountID = "accountId"
	FieldOwnerID   = "sb_owner"
	FieldToken     = "token"
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
