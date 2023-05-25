package mongo

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (mg *Mongo) AddFormSubmission(dbName, form string, doc map[string]interface{}) error {
	db := mg.Client.Database(dbName)

	doc[FieldID] = primitive.NewObjectID()
	doc[FieldFormName] = form
	doc["sb_posted"] = time.Now()

	if _, err := db.Collection("sb_forms").InsertOne(mg.Ctx, doc); err != nil {
		return err
	}
	return nil
}

func (mg *Mongo) ListFormSubmissions(dbName, name string) (results []map[string]interface{}, err error) {
	db := mg.Client.Database(dbName)

	opt := options.Find()
	opt.SetLimit(100)
	opt.SetSort(bson.M{FieldID: -1})

	filter := bson.M{}
	if len(name) > 0 {
		filter["form"] = name
	}

	cur, err := db.Collection("sb_forms").Find(mg.Ctx, filter, opt)
	if err != nil {
		return
	}
	defer cur.Close(mg.Ctx)

	for cur.Next(mg.Ctx) {
		var result bson.M
		if err := cur.Decode(&result); err != nil {
			return nil, err
		}

		result["id"] = result[FieldID]
		delete(result, FieldID)

		results = append(results, result)
	}
	if err = cur.Err(); err != nil {
		return
	}

	if len(results) == 0 {
		results = make([]map[string]interface{}, 1)
	}

	return
}

func (mg *Mongo) GetForms(dbName string) ([]string, error) {
	db := mg.Client.Database(dbName)

	// previously was bson.D{{"$group", bson.D{"_id", "$form"}}}
	pipeline := mongo.Pipeline{bson.D{
		{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$form"},
		}}}}
	cur, err := db.Collection("sb_forms").Aggregate(mg.Ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(mg.Ctx)

	var names []string
	for cur.Next(mg.Ctx) {
		var form bson.M
		if err := cur.Decode(&form); err != nil {
			return nil, err
		}

		names = append(names, fmt.Sprintf("%v", form[FieldID]))
	}

	if err := cur.Err(); err != nil {
		return nil, err
	}

	if len(names) == 0 {
		names = make([]string, 1)
	}

	return names, nil
}
