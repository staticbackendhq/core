package mongo

import (
	"errors"
	"time"

	"github.com/staticbackendhq/core/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type LocalAccount struct {
	ID      primitive.ObjectID `bson:"_id" json:"id"`
	Email   string             `bson:"email" json:"email"`
	Created time.Time          `bson:"created" json:"created"`
}

func fromLocalAccount(a LocalAccount) model.Account {
	return model.Account{
		ID:      a.ID.Hex(),
		Email:   a.Email,
		Created: a.Created,
	}
}

func (mg *Mongo) CreateAccount(dbName, email string) (id string, err error) {
	db := mg.Client.Database(dbName)

	a := LocalAccount{
		ID:    primitive.NewObjectID(),
		Email: email,
	}

	_, err = db.Collection("sb_accounts").InsertOne(mg.Ctx, a)
	if err != nil {
		return
	}

	id = a.ID.Hex()
	return
}

func (mg *Mongo) CreateUser(dbName string, tok model.User) (id string, err error) {
	db := mg.Client.Database(dbName)

	tok.Created = time.Now()

	tok.ID = primitive.NewObjectID().Hex()

	itok := toLocalToken(tok)

	_, err = db.Collection("sb_tokens").InsertOne(mg.Ctx, itok)
	if err != nil {
		return
	}

	id = tok.ID
	return
}

func (mg *Mongo) UserEmailExists(dbName, email string) (exists bool, err error) {
	db := mg.Client.Database(dbName)

	count, err := db.Collection("sb_tokens").CountDocuments(mg.Ctx, bson.M{"email": email})
	if err != nil {
		return
	}

	exists = count > 0
	return
}

func (mg *Mongo) SetUserRole(dbName, email string, role int) error {
	db := mg.Client.Database(dbName)

	filter := bson.M{"email": email}
	update := bson.M{"$set": bson.M{"role": role}}
	if _, err := db.Collection("sb_tokens").UpdateOne(mg.Ctx, filter, update); err != nil {
		return err
	}
	return nil
}

func (mg *Mongo) UserSetPassword(dbName, tokenID, password string) error {
	db := mg.Client.Database(dbName)

	id, err := primitive.ObjectIDFromHex(tokenID)
	if err != nil {
		return err
	}

	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"pw": password}}
	if _, err := db.Collection("sb_tokens").UpdateOne(mg.Ctx, filter, update); err != nil {
		return err
	}
	return nil
}

func (mg *Mongo) GetFirstUserFromAccountID(dbName, accountID string) (tok model.User, err error) {
	db := mg.Client.Database(dbName)

	oid, err := primitive.ObjectIDFromHex(accountID)
	if err != nil {
		return
	}

	filter := bson.M{FieldAccountID: oid}

	opt := options.Find()
	opt.SetLimit(1)
	opt.SetSort(bson.M{FieldID: 1})

	cur, err := db.Collection("sb_tokens").Find(mg.Ctx, filter, opt)
	if err != nil {
		return
	}
	defer cur.Close(mg.Ctx)

	var lt LocalToken
	if cur.Next(mg.Ctx) {
		if err = cur.Decode(&lt); err != nil {
			return
		}
	}

	tok = fromLocalToken(lt)

	if len(tok.Token) == 0 {
		return tok, errors.New("invalid account id")
	}

	return
}

func (mg *Mongo) RemoveUser(auth model.Auth, dbName, userID string) error {
	db := mg.Client.Database(dbName)

	aid, err := primitive.ObjectIDFromHex(auth.AccountID)
	if err != nil {
		return err
	}

	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	filter := bson.M{FieldID: uid, FieldAccountID: aid}
	if _, err := db.Collection("sb_tokens").DeleteOne(mg.Ctx, filter); err != nil {
		return err
	}
	return nil
}
