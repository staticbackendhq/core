package mongo

import (
	"errors"
	"time"

	"github.com/staticbackendhq/core/model"
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
	FieldFormName  = "form"
)

type LocalToken struct {
	ID        primitive.ObjectID `bson:"_id" json:"id"`
	AccountID primitive.ObjectID `bson:"accountId" json:"accountId"`
	Token     string             `bson:"token" json:"token"`
	Email     string             `bson:"email" json:"email"`
	Password  string             `bson:"pw" json:"-"`
	Role      int                `bson:"role" json:"role"`
	ResetCode string             `bson:"resetCode" json:"-"`
	Created   time.Time          `bson:"created" json:"created"`
}

func toLocalToken(token model.User) LocalToken {
	id, err := primitive.ObjectIDFromHex(token.ID)
	if err != nil {
		return LocalToken{}
	}

	acctID, err := primitive.ObjectIDFromHex(token.AccountID)
	if err != nil {
		return LocalToken{}
	}

	return LocalToken{
		ID:        id,
		AccountID: acctID,
		Token:     token.Token,
		Email:     token.Email,
		Password:  token.Password,
		Role:      token.Role,
		ResetCode: token.ResetCode,
		Created:   token.Created,
	}
}

func fromLocalToken(tok LocalToken) model.User {
	return model.User{
		ID:        tok.ID.Hex(),
		AccountID: tok.AccountID.Hex(),
		Token:     tok.Token,
		Email:     tok.Email,
		Password:  tok.Password,
		Role:      tok.Role,
		ResetCode: tok.ResetCode,
		Created:   tok.Created,
	}
}

func (mg *Mongo) FindUser(dbName, userID, token string) (tok model.User, err error) {
	db := mg.Client.Database(dbName)

	id, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return
	}

	var lt LocalToken
	sr := db.Collection("sb_tokens").FindOne(mg.Ctx, bson.M{FieldID: id, FieldToken: token})
	err = sr.Decode(&lt)

	tok = fromLocalToken(lt)
	return
}

func (mg *Mongo) FindRootUser(dbName, userID, accountID, token string) (tok model.User, err error) {
	db := mg.Client.Database(dbName)

	id, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return
	}

	acctID, err := primitive.ObjectIDFromHex(accountID)
	if err != nil {
		return
	}

	filter := bson.M{
		FieldID:        id,
		FieldAccountID: acctID,
		FieldToken:     token,
	}

	var lt LocalToken

	sr := db.Collection("sb_tokens").FindOne(mg.Ctx, filter)
	err = sr.Decode(&lt)

	tok = fromLocalToken(lt)

	return
}

func (mg *Mongo) GetRootForBase(dbName string) (tok model.User, err error) {
	db := mg.Client.Database(dbName)

	filter := bson.M{
		FieldRole: 100,
	}

	var lt LocalToken

	sr := db.Collection("sb_tokens").FindOne(mg.Ctx, filter)
	err = sr.Decode(&lt)

	tok = fromLocalToken(lt)

	return
}

func (mg *Mongo) ListAccounts(dbName string) ([]model.Account, error) {
	db := mg.Client.Database(dbName)

	cur, err := db.Collection("sb_accounts").Find(mg.Ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(mg.Ctx)

	var list []model.Account

	for cur.Next(mg.Ctx) {
		var v LocalAccount
		err := cur.Decode(&v)
		if err != nil {
			return nil, err
		}

		list = append(list, fromLocalAccount(v))
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

func (mg *Mongo) ListUsers(dbName, accountID string) ([]model.User, error) {
	db := mg.Client.Database(dbName)

	id, err := primitive.ObjectIDFromHex(accountID)
	if err != nil {
		return nil, err
	}

	filter := bson.M{FieldAccountID: id}

	cur, err := db.Collection("sb_tokens").Find(mg.Ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cur.Close(mg.Ctx)

	var list []model.User

	for cur.Next(mg.Ctx) {
		var v LocalToken
		err := cur.Decode(&v)
		if err != nil {
			return nil, err
		}

		list = append(list, fromLocalToken(v))
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

func (mg *Mongo) FindUserByEmail(dbName, email string) (tok model.User, err error) {
	db := mg.Client.Database(dbName)

	var lt LocalToken

	sr := db.Collection("sb_tokens").FindOne(mg.Ctx, bson.M{"email": email})
	err = sr.Decode(&lt)

	tok = fromLocalToken(lt)

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

func (mg *Mongo) GetUserByID(dbName, accountID, userID string) (user model.User, err error) {
	db := mg.Client.Database(dbName)

	aid, err := primitive.ObjectIDFromHex(accountID)
	if err != nil {
		return
	}

	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return
	}

	filter := bson.M{FieldID: uid, FieldAccountID: aid}
	sr := db.Collection("sb_tokens").FindOne(mg.Ctx, filter)
	err = sr.Decode(&user)
	return
}
