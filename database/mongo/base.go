package mongo

import (
	"fmt"
	"strings"

	"github.com/staticbackendhq/core/internal"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Base struct {
	PublishDocument func(topic, msg string, doc interface{})
}

func (mg *Mongo) CreateDocument(auth internal.Auth, dbName, col string, doc map[string]interface{}) (map[string]interface{}, error) {
	db := mg.Client.Database(dbName)

	delete(doc, "id")
	delete(doc, FieldID)
	delete(doc, FieldAccountID)
	delete(doc, FieldOwnerID)

	acctID, userID, err := parseObjectID(auth)
	if err != nil {
		return nil, err
	}

	doc[FieldID] = primitive.NewObjectID()
	doc[FieldAccountID] = acctID
	doc[FieldOwnerID] = userID

	if _, err := db.Collection(col).InsertOne(mg.Ctx, doc); err != nil {
		return nil, err
	}

	doc["id"] = doc[FieldID]
	delete(doc, FieldID)

	// TODONOW: publish this
	//b.PublishDocument("db-"+col, internal.MsgTypeDBCreated, doc)

	return doc, nil
}

func (mg *Mongo) BulkCreateDocument(auth internal.Auth, dbName, col string, docs []interface{}) error {
	db := mg.Client.Database(dbName)

	acctID, userID, err := parseObjectID(auth)
	if err != nil {
		return err
	}

	for _, item := range docs {
		doc, ok := item.(map[string]interface{})
		if !ok {
			return fmt.Errorf("unable to cast docs to map")
		}

		delete(doc, "id")
		delete(doc, FieldID)
		delete(doc, FieldAccountID)
		delete(doc, FieldOwnerID)

		doc[FieldID] = primitive.NewObjectID()
		doc[FieldAccountID] = acctID
		doc[FieldOwnerID] = userID
	}

	if _, err := db.Collection(col).InsertMany(mg.Ctx, docs); err != nil {
		return err
	}
	return nil
}

func (mg *Mongo) ListDocuments(auth internal.Auth, dbName, col string, params internal.ListParams) (internal.PagedResult, error) {
	db := mg.Client.Database(dbName)

	result := internal.PagedResult{
		Page: params.Page,
		Size: params.Size,
	}

	acctID, userID, err := parseObjectID(auth)
	if err != nil {
		return result, err
	}

	filter := bson.M{}

	// if they're not root
	if !strings.HasPrefix(col, "pub_") && auth.Role < 100 {
		switch internal.ReadPermission(col) {
		case internal.PermGroup:
			filter = bson.M{FieldAccountID: acctID}
		case internal.PermOwner:
			filter = bson.M{FieldAccountID: auth.AccountID, FieldOwnerID: userID}
		}
	}

	count, err := db.Collection(col).CountDocuments(mg.Ctx, filter)
	if err != nil {
		return result, err
	}

	result.Total = count

	skips := params.Size * (params.Page - 1)

	if len(params.SortBy) == 0 || strings.EqualFold(params.SortBy, "id") {
		params.SortBy = FieldID
	}
	sortBy := bson.M{params.SortBy: 1}
	if params.SortDescending {
		sortBy[params.SortBy] = -1
	}

	opt := options.Find()
	opt.SetSkip(skips)
	opt.SetLimit(params.Size)
	opt.SetSort(sortBy)

	cur, err := db.Collection(col).Find(mg.Ctx, filter, opt)
	if err != nil {
		return result, err
	}
	defer cur.Close(mg.Ctx)

	var results []map[string]interface{}

	for cur.Next(mg.Ctx) {
		var v map[string]interface{}
		err := cur.Decode(&v)
		if err != nil {
			return result, err
		}

		v["id"] = v[FieldID]
		delete(v, FieldID)
		delete(v, FieldOwnerID)

		results = append(results, v)
	}
	if err := cur.Err(); err != nil {
		return result, err
	}

	if len(results) == 0 {
		results = make([]map[string]interface{}, 0)
	}

	result.Results = results

	return result, nil
}

func (mg *Mongo) QueryDocuments(auth internal.Auth, dbName, col string, filter map[string]interface{}, params internal.ListParams) (internal.PagedResult, error) {
	db := mg.Client.Database(dbName)

	result := internal.PagedResult{
		Page: params.Page,
		Size: params.Size,
	}

	acctID, userID, err := parseObjectID(auth)
	if err != nil {
		return result, err
	}

	// either not a public repo or not root
	if strings.HasPrefix(col, "pub_") == false && auth.Role < 100 {
		switch internal.ReadPermission(col) {
		case internal.PermGroup:
			filter[FieldAccountID] = acctID
		case internal.PermOwner:
			filter[FieldAccountID] = acctID
			filter[FieldOwnerID] = userID
		}

	}

	count, err := db.Collection(col).CountDocuments(mg.Ctx, filter)
	if err != nil {
		return result, err
	}

	result.Total = count

	if count == 0 {
		result.Results = make([]map[string]interface{}, 0)
		return result, nil
	}

	skips := params.Size * (params.Page - 1)

	if len(params.SortBy) == 0 || strings.EqualFold(params.SortBy, "id") {
		params.SortBy = FieldID
	}
	sortBy := bson.M{params.SortBy: 1}
	if params.SortDescending {
		sortBy[params.SortBy] = -1
	}

	opt := options.Find()
	opt.SetSkip(skips)
	opt.SetLimit(params.Size)
	opt.SetSort(sortBy)

	cur, err := db.Collection(col).Find(mg.Ctx, filter, opt)
	if err != nil {
		return result, err
	}
	defer cur.Close(mg.Ctx)

	var results []map[string]interface{}
	for cur.Next(mg.Ctx) {
		var v map[string]interface{}
		if err := cur.Decode(&v); err != nil {
			return result, err
		}

		v["id"] = v[FieldID]
		delete(v, FieldID)
		delete(v, FieldOwnerID)

		results = append(results, v)
	}

	if err := cur.Err(); err != nil {
		return result, err
	}

	result.Results = results

	return result, nil
}

func (mg *Mongo) GetDocumentByID(auth internal.Auth, dbName, col, id string) (map[string]interface{}, error) {
	db := mg.Client.Database(dbName)

	var result map[string]interface{}

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return result, err
	}

	acctID, userID, err := parseObjectID(auth)
	if err != nil {
		return result, err
	}

	filter := bson.M{FieldID: oid}

	// if they're not root and repo is not public
	if !strings.HasPrefix(col, "pub_") && auth.Role < 100 {
		switch internal.ReadPermission(col) {
		case internal.PermGroup:
			filter[FieldAccountID] = acctID
		case internal.PermOwner:
			filter[FieldAccountID] = acctID
			filter[FieldOwnerID] = userID
		}
	}

	sr := db.Collection(col).FindOne(mg.Ctx, filter)
	if err := sr.Decode(&result); err != nil {
		return result, err
	} else if err := sr.Err(); err != nil {
		return result, err
	}

	result["id"] = result[FieldID]
	delete(result, FieldID)
	delete(result, FieldOwnerID)

	return result, nil
}

func (mg *Mongo) UpdateDocument(auth internal.Auth, dbName, col, id string, doc map[string]interface{}) (map[string]interface{}, error) {
	db := mg.Client.Database(dbName)

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return doc, err
	}

	acctID, userID, err := parseObjectID(auth)
	if err != nil {
		return nil, err
	}

	delete(doc, "id")
	delete(doc, FieldID)
	delete(doc, FieldAccountID)
	delete(doc, FieldOwnerID)

	filter := bson.M{FieldID: oid}

	// if they are not "root", we use permission
	if auth.Role < 100 {
		switch internal.WritePermission(col) {
		case internal.PermGroup:
			filter[FieldAccountID] = acctID
		case internal.PermOwner:
			filter[FieldAccountID] = acctID
			filter[FieldOwnerID] = userID
		}
	}

	newProps := bson.M{}
	for k, v := range doc {
		newProps[k] = v
	}

	update := bson.M{"$set": newProps}

	res := db.Collection(col).FindOneAndUpdate(mg.Ctx, filter, update)
	if err := res.Err(); err != nil {
		return doc, err
	}

	var result bson.M
	sr := db.Collection(col).FindOne(mg.Ctx, filter)
	if err := sr.Decode(&result); err != nil {
		return doc, err
	} else if err := sr.Err(); err != nil {
		return doc, err
	}

	result["id"] = result[FieldID]
	delete(result, FieldID)
	delete(result, FieldOwnerID)

	//TODONOW: publish db event
	//b.PublishDocument("db-"+col, internal.MsgTypeDBUpdated, result)

	return result, nil
}

func (mg *Mongo) IncrementValue(auth internal.Auth, dbName, col, id, field string, n int) error {
	db := mg.Client.Database(dbName)

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	acctID, userID, err := parseObjectID(auth)
	if err != nil {
		return err
	}

	filter := bson.M{FieldID: oid}

	// if they are not "root", we use permission
	if auth.Role < 100 {
		switch internal.WritePermission(col) {
		case internal.PermGroup:
			filter[FieldAccountID] = acctID
		case internal.PermOwner:
			filter[FieldAccountID] = acctID
			filter[FieldOwnerID] = userID
		}
	}

	update := bson.M{"$inc": bson.M{field: n}}

	res := db.Collection(col).FindOneAndUpdate(mg.Ctx, filter, update)
	if err := res.Err(); err != nil {
		return err
	}

	//TODONOW: publish db event
	/*
		var result bson.M
		sr := db.Collection(col).FindOne(ctx, filter)
		if err := sr.Decode(&result); err != nil {
			return err
		} else if err := sr.Err(); err != nil {
			return err
		}

		result["id"] = result[internal.FieldID]
		delete(result, internal.FieldID)
		delete(result, internal.FieldOwnerID)

		b.PublishDocument("db-"+col, internal.MsgTypeDBUpdated, result)
	*/

	return nil
}

func (mg *Mongo) DeleteDocument(auth internal.Auth, dbName, col, id string) (int64, error) {
	db := mg.Client.Database(dbName)

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return 0, err
	}

	acctID, userID, err := parseObjectID(auth)
	if err != nil {
		return 0, err
	}

	filter := bson.M{FieldID: oid}

	// if they're not root
	if auth.Role < 100 {
		switch internal.WritePermission(col) {
		case internal.PermGroup:
			filter[FieldAccountID] = acctID
		case internal.PermOwner:
			filter[FieldAccountID] = acctID
			filter[FieldOwnerID] = userID
		}
	}

	res, err := db.Collection(col).DeleteOne(mg.Ctx, filter)
	if err != nil {
		return 0, err
	}

	//TODONOW: public db event
	//b.PublishDocument("db-"+col, internal.MsgTypeDBDeleted, id)

	return res.DeletedCount, nil
}

func (mg *Mongo) ListCollections(dbName string) ([]string, error) {
	db := mg.Client.Database(dbName)

	cur, err := db.ListCollections(mg.Ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(mg.Ctx)

	var names []string
	for cur.Next(mg.Ctx) {
		var result bson.M
		err := cur.Decode(&result)
		if err != nil {
			return nil, err
		}

		names = append(names, fmt.Sprintf("%v", result["name"]))
	}

	return names, nil
}

func parseObjectID(auth internal.Auth) (acctID, userID primitive.ObjectID, err error) {
	acctID, err = primitive.ObjectIDFromHex(auth.AccountID)
	if err != nil {
		return
	}
	userID, err = primitive.ObjectIDFromHex(auth.UserID)
	return
}
