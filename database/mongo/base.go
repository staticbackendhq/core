package mongo

import (
	"fmt"
	"strings"
	"sync"

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

	newID := primitive.NewObjectID()

	doc[FieldID] = newID
	doc[FieldAccountID] = acctID
	doc[FieldOwnerID] = userID

	if _, err := db.Collection(internal.CleanCollectionName(col)).InsertOne(mg.Ctx, doc); err != nil {
		return nil, err
	}

	cleanMap(doc)

	mg.PublishDocument("db-"+col, internal.MsgTypeDBCreated, doc)

	go mg.ensureIndex(dbName, internal.CleanCollectionName(col))

	return doc, nil
}

var (
	checkIndex = make(map[string]bool)
	mutx       = sync.RWMutex{}
)

func (mg *Mongo) ensureIndex(dbName, col string) {
	key := fmt.Sprintf("%s_%s", dbName, col)

	mutx.RLock()
	if _, ok := checkIndex[key]; ok {
		return
	}
	mutx.RUnlock()

	db := mg.Client.Database(dbName)

	dbCol := db.Collection(col)

	cur, err := dbCol.Indexes().List(mg.Ctx)
	if err != nil {
		mg.log.Warn().Err(err).Msg("error getting col indexes")

		return
	}

	found := false
	for cur.Next(mg.Ctx) {
		var v bson.M
		if err := cur.Decode(&v); err != nil {
			mg.log.Warn().Err(err).Msg("cannot cast to IndexModel")
			return
		}

		keys, ok := v["key"].(bson.M)
		if !ok {
			mg.log.Warn().Msg("unable to cast IndexModel Key to map")

			return
		}

		for k := range keys {
			if k == "accountId" {
				found = true
				break
			}
		}

		if found {
			break
		}
	}

	mutx.Lock()
	checkIndex[key] = true
	mutx.Unlock()

	if found {
		return
	}

	if err := mg.CreateIndex(dbName, col, FieldAccountID); err != nil {
		mg.log.Warn().Err(err).Msg("error creating accountId idx")
	}
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

	if _, err := db.Collection(internal.CleanCollectionName(col)).InsertMany(mg.Ctx, docs); err != nil {
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

	secureRead(acctID, userID, auth.Role, col, filter)

	count, err := db.Collection(internal.CleanCollectionName(col)).CountDocuments(mg.Ctx, filter)
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

	cur, err := db.Collection(internal.CleanCollectionName(col)).Find(mg.Ctx, filter, opt)
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

		cleanMap(v)

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

	secureRead(acctID, userID, auth.Role, col, filter)

	count, err := db.Collection(internal.CleanCollectionName(col)).CountDocuments(mg.Ctx, filter)
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

	cur, err := db.Collection(internal.CleanCollectionName(col)).Find(mg.Ctx, filter, opt)
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

		cleanMap(v)

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

	secureRead(acctID, userID, auth.Role, col, filter)

	sr := db.Collection(internal.CleanCollectionName(col)).FindOne(mg.Ctx, filter)
	if err := sr.Decode(&result); err != nil {
		return result, err
	} else if err := sr.Err(); err != nil {
		return result, err
	}

	cleanMap(result)

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

	removeNotEditableFields(doc)

	filter := bson.M{FieldID: oid}

	secureWrite(acctID, userID, auth.Role, col, filter)

	newProps := bson.M{}
	for k, v := range doc {
		newProps[k] = v
	}

	update := bson.M{"$set": newProps}

	res := db.Collection(internal.CleanCollectionName(col)).FindOneAndUpdate(mg.Ctx, filter, update)
	if err := res.Err(); err != nil {
		return doc, err
	}

	var result bson.M
	sr := db.Collection(internal.CleanCollectionName(col)).FindOne(mg.Ctx, filter)
	if err := sr.Decode(&result); err != nil {
		return doc, err
	} else if err := sr.Err(); err != nil {
		return doc, err
	}

	cleanMap(result)

	mg.PublishDocument("db-"+col, internal.MsgTypeDBUpdated, result)

	return result, nil
}

func (mg *Mongo) UpdateDocuments(auth internal.Auth, dbName, col string, filters map[string]interface{}, updateFields map[string]interface{}) (n int64, err error) {
	db := mg.Client.Database(dbName)

	acctID, userID, err := parseObjectID(auth)
	if err != nil {
		return 0, err
	}

	secureWrite(acctID, userID, auth.Role, col, filters)
	removeNotEditableFields(updateFields)

	newProps := bson.M{}
	for k, v := range updateFields {
		newProps[k] = v
	}

	update := bson.M{"$set": newProps}

	res, err := db.Collection(internal.CleanCollectionName(col)).UpdateMany(mg.Ctx, filters, update)
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, err
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

	secureWrite(acctID, userID, auth.Role, col, filter)

	update := bson.M{"$inc": bson.M{field: n}}

	res := db.Collection(internal.CleanCollectionName(col)).FindOneAndUpdate(mg.Ctx, filter, update)
	if err := res.Err(); err != nil {
		return err
	}

	updated, err := mg.GetDocumentByID(auth, dbName, col, id)
	if err != nil {
		return err
	}

	mg.PublishDocument("db-"+col, internal.MsgTypeDBUpdated, updated)

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

	secureWrite(acctID, userID, auth.Role, col, filter)

	res, err := db.Collection(internal.CleanCollectionName(col)).DeleteOne(mg.Ctx, filter)
	if err != nil {
		return 0, err
	}

	mg.PublishDocument("db-"+col, internal.MsgTypeDBDeleted, id)

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

func cleanMap(m map[string]interface{}) {
	oid, ok := m[FieldID].(primitive.ObjectID)
	if !ok {
		return
	}

	m["id"] = oid.Hex()
	delete(m, FieldID)

	oid, ok = m[FieldAccountID].(primitive.ObjectID)
	if !ok {
		return
	}

	m[FieldAccountID] = oid.Hex()

	delete(m, FieldOwnerID)
}

func removeNotEditableFields(m map[string]any) {
	delete(m, "id")
	delete(m, FieldID)
	delete(m, FieldAccountID)
	delete(m, FieldOwnerID)
}
