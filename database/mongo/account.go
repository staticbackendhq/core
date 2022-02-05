package mongo

import (
	"errors"

	"github.com/staticbackendhq/core/internal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	FieldID        = "_id"
	FieldAccountID = "accountId"
	FieldOwnerID   = "sb_owner"
	FieldToken     = "token"
	FieldIsActive  = "active"
	FieldRole      = "role"
)

func (mg *Mongo) FindToken(dbName, tokenID, token string) (tok internal.Token, err error) {
	db := mg.Client.Database(dbName)

	id, err := primitive.NewObjectIDFromHex(tokenID)
	if err != nil {
		return
	}

	sr := db.Collection("sb_tokens").FindOne(mg.Ctx, bson.M{FieldID: id, FieldToken: token})
	err = sr.Decode(&tok)
	return
}

func (mg *Mongo) FindRootToken(dbName, tokenID, accountID, token string) (tok internal.Token, err error) {
	db := mg.Client.Database(dbName)

	id, err := primitive.NewObjectIDFromHex(tokenID)
	if err != nil {
		return
	}

	acctID, err := primitive.NewObjectIDFromHex(accountID)
	if err != nil {
		return
	}

	filter := bson.M{
		FieldID:        id,
		FieldAccountID: acctID,
		FieldToken:     token,
	}
	sr := db.Collection("sb_tokens").FindOne(mg.Ctx, filter)
	err = sr.Decode(&tok)
	return
}

func (mg *Mongo) GetRootForBase(dbName string) (tok internal.Token, err error) {
	db := mg.Client.Database(dbName)

	filter := bson.M{
		FieldRole: 100,
	}
	sr := db.Collection("sb_tokens").FindOne(mg.Ctx, filter)
	err = sr.Decode(&tok)
	return
}

func (mg *Mongo) FindTokenByEmail(dbName, email string) (tok internal.Token, err error) {
	db := mg.Client.Database(dbName)

	sr := db.Collection("sb_tokens").FindOne(mg.Ctx, bson.M{"email": email})
	err = sr.Decode(&tok)
	return
}

func (mg *Mongo) SetPasswordResetCode(dbName, tokenID, code string) error {
	db := mg.Client.Database(dbName)

	id, err := primitive.ObjectIDFromHex(tokenID)
	if err != nil {
		return err
	}

	update := bson.M{"$set": bson.M{"resetCode": code}}
	if _, err := db.Collection("sb_tokens").UpdateByID(mg.Ctx, id, update); err != nil {
		return err
	}
	return nil
}

func (mg *Mongo) ResetPassword(dbName, email, code, password string) error {
	db := mg.Client.Database(dbName)

	filter := bson.M{"email": email, "resetCode": code}
	update := bson.M{"$set": bson.M{"pw": password}}
	res, err := db.Collection("sb_tokens").UpdateOne(mg.Ctx, filter, update)
	if err != nil {
		return err
	} else if res.ModifiedCount != 1 {
		return errors.New("cannot find document")
	}
	return nil
}
