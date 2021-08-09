package db

import (
	"context"
	"fmt"
	"staticbackend/internal"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Base struct {
	PublishDocument func(topic, msg string, doc interface{})
}

func (b *Base) Add(auth internal.Auth, db *mongo.Database, col string, doc map[string]interface{}) (map[string]interface{}, error) {
	delete(doc, "id")
	delete(doc, internal.FieldID)
	delete(doc, internal.FieldAccountID)
	delete(doc, internal.FieldOwnerID)

	doc[internal.FieldID] = primitive.NewObjectID()
	doc[internal.FieldAccountID] = auth.AccountID
	doc[internal.FieldOwnerID] = auth.UserID

	ctx := context.Background()
	if _, err := db.Collection(col).InsertOne(ctx, doc); err != nil {
		return nil, err
	}

	doc["id"] = doc[internal.FieldID]
	delete(doc, internal.FieldID)

	b.PublishDocument("db-"+col, internal.MsgTypeDBCreated, doc)

	return doc, nil
}

type PagedResult struct {
	Page    int64    `json:"page"`
	Size    int64    `json:"size"`
	Total   int64    `json:"total"`
	Results []bson.M `json:"results"`
}

type ListParams struct {
	Page           int64
	Size           int64
	SortBy         string
	SortDescending bool
}

func (b *Base) List(auth internal.Auth, db *mongo.Database, col string, params ListParams) (PagedResult, error) {
	result := PagedResult{
		Page: params.Page,
		Size: params.Size,
	}

	filter := bson.M{}

	// if they're not root
	if !strings.HasPrefix(col, "pub_") && auth.Role < 100 {
		switch internal.ReadPermission(col) {
		case internal.PermGroup:
			filter = bson.M{internal.FieldAccountID: auth.AccountID}
		case internal.PermOwner:
			filter = bson.M{internal.FieldAccountID: auth.AccountID, internal.FieldOwnerID: auth.UserID}
		}
	}

	ctx := context.Background()
	count, err := db.Collection(col).CountDocuments(ctx, filter)
	if err != nil {
		return result, err
	}

	result.Total = count

	skips := params.Size * (params.Page - 1)

	if len(params.SortBy) == 0 || strings.EqualFold(params.SortBy, "id") {
		params.SortBy = internal.FieldID
	}
	sortBy := bson.M{params.SortBy: 1}
	if params.SortDescending {
		sortBy[params.SortBy] = -1
	}

	opt := options.Find()
	opt.SetSkip(skips)
	opt.SetLimit(params.Size)
	opt.SetSort(sortBy)

	cur, err := db.Collection(col).Find(ctx, filter, opt)
	if err != nil {
		return result, err
	}
	defer cur.Close(ctx)

	var results []bson.M

	for cur.Next(ctx) {
		var v bson.M
		err := cur.Decode(&v)
		if err != nil {
			return result, err
		}

		v["id"] = v[internal.FieldID]
		delete(v, internal.FieldID)
		delete(v, internal.FieldOwnerID)

		results = append(results, v)
	}
	if err := cur.Err(); err != nil {
		return result, err
	}

	if len(results) == 0 {
		results = make([]bson.M, 1)
	}

	result.Results = results

	return result, nil
}

func (b *Base) Query(auth internal.Auth, db *mongo.Database, col string, filter bson.M, params ListParams) (PagedResult, error) {
	result := PagedResult{
		Page: params.Page,
		Size: params.Size,
	}

	// either not a public repo or not root
	if strings.HasPrefix(col, "pub_") == false && auth.Role < 100 {
		switch internal.ReadPermission(col) {
		case internal.PermGroup:
			filter[internal.FieldAccountID] = auth.AccountID
		case internal.PermOwner:
			filter[internal.FieldAccountID] = auth.AccountID
			filter[internal.FieldOwnerID] = auth.UserID
		}

	}

	ctx := context.Background()
	count, err := db.Collection(col).CountDocuments(ctx, filter)
	if err != nil {
		return result, err
	}

	result.Total = count

	if count == 0 {
		result.Results = make([]bson.M, 0)
		return result, nil
	}

	skips := params.Size * (params.Page - 1)

	if len(params.SortBy) == 0 || strings.EqualFold(params.SortBy, "id") {
		params.SortBy = internal.FieldID
	}
	sortBy := bson.M{params.SortBy: 1}
	if params.SortDescending {
		sortBy[params.SortBy] = -1
	}

	opt := options.Find()
	opt.SetSkip(skips)
	opt.SetLimit(params.Size)
	opt.SetSort(sortBy)

	cur, err := db.Collection(col).Find(ctx, filter, opt)
	if err != nil {
		return result, err
	}
	defer cur.Close(ctx)

	var results []bson.M

	for cur.Next(ctx) {
		var v bson.M
		if err := cur.Decode(&v); err != nil {
			return result, err
		}

		v["id"] = v[internal.FieldID]
		delete(v, internal.FieldID)
		delete(v, internal.FieldOwnerID)

		results = append(results, v)
	}

	if err := cur.Err(); err != nil {
		return result, err
	}

	if len(results) == 0 {
		results = make([]bson.M, 1)
	}

	result.Results = results

	return result, nil
}

func (b *Base) GetByID(auth internal.Auth, db *mongo.Database, col, id string) (bson.M, error) {
	var result bson.M

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return result, err
	}

	filter := bson.M{internal.FieldID: oid}

	// if they're not root and repo is not public
	if !strings.HasPrefix(col, "pub_") && auth.Role < 100 {
		switch internal.ReadPermission(col) {
		case internal.PermGroup:
			filter[internal.FieldAccountID] = auth.AccountID
		case internal.PermOwner:
			filter[internal.FieldAccountID] = auth.AccountID
			filter[internal.FieldOwnerID] = auth.UserID
		}
	}

	ctx := context.Background()
	sr := db.Collection(col).FindOne(ctx, filter)
	if err := sr.Decode(&result); err != nil {
		return result, err
	} else if err := sr.Err(); err != nil {
		return result, err
	}

	result["id"] = result[internal.FieldID]
	delete(result, internal.FieldID)
	delete(result, internal.FieldOwnerID)

	return result, nil
}

func (b *Base) Update(auth internal.Auth, db *mongo.Database, col, id string, doc map[string]interface{}) (map[string]interface{}, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return doc, err
	}

	delete(doc, "id")
	delete(doc, internal.FieldID)
	delete(doc, internal.FieldAccountID)
	delete(doc, internal.FieldOwnerID)

	filter := bson.M{internal.FieldID: oid}

	// if they are not "root", we use permission
	if auth.Role < 100 {
		switch internal.WritePermission(col) {
		case internal.PermGroup:
			filter[internal.FieldAccountID] = auth.AccountID
		case internal.PermOwner:
			filter[internal.FieldAccountID] = auth.AccountID
			filter[internal.FieldOwnerID] = auth.UserID
		}
	}

	newProps := bson.M{}
	for k, v := range doc {
		newProps[k] = v
	}

	update := bson.M{"$set": newProps}

	ctx := context.Background()
	res := db.Collection(col).FindOneAndUpdate(ctx, filter, update)
	if err := res.Err(); err != nil {
		return doc, err
	}

	var result bson.M
	sr := db.Collection(col).FindOne(ctx, filter)
	if err := sr.Decode(&result); err != nil {
		return doc, err
	} else if err := sr.Err(); err != nil {
		return doc, err
	}

	result["id"] = result[internal.FieldID]
	delete(result, internal.FieldID)
	delete(result, internal.FieldOwnerID)

	b.PublishDocument("db-"+col, internal.MsgTypeDBUpdated, result)

	return result, nil
}

func (b *Base) Delete(auth internal.Auth, db *mongo.Database, col, id string) (int64, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return 0, err
	}

	filter := bson.M{internal.FieldID: oid}

	// if they're not root
	if auth.Role < 100 {
		switch internal.WritePermission(col) {
		case internal.PermGroup:
			filter[internal.FieldAccountID] = auth.AccountID
		case internal.PermOwner:
			filter[internal.FieldAccountID] = auth.AccountID
			filter[internal.FieldOwnerID] = auth.UserID

		}
	}
	ctx := context.Background()
	res, err := db.Collection(col).DeleteOne(ctx, filter)
	if err != nil {
		return 0, err
	}

	b.PublishDocument("db-"+col, internal.MsgTypeDBDeleted, id)

	return res.DeletedCount, nil
}

func (b *Base) ListCollections(db *mongo.Database) ([]string, error) {
	ctx := context.Background()

	cur, err := db.ListCollections(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var names []string
	for cur.Next(ctx) {
		var result bson.M
		err := cur.Decode(&result)
		if err != nil {
			return nil, err
		}

		names = append(names, fmt.Sprintf("%v", result["name"]))
	}

	return names, nil
}
