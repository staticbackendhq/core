package internal

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ListFormSubmissions(db *mongo.Database, name string) ([]bson.M, error) {
	opt := options.Find()
	opt.SetLimit(100)
	opt.SetSort(bson.M{FieldID: -1})

	filter := bson.M{}
	if len(name) > 0 {
		filter["form"] = name
	}

	cur, err := db.Collection("sb_forms").Find(ctx, filter, opt)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var results []bson.M
	for cur.Next(ctx) {
		var result bson.M
		if err := cur.Decode(&result); err != nil {
			return nil, err
		}

		result["id"] = result[FieldID]
		delete(result, FieldID)

		results = append(results, result)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		results = make([]bson.M, 1)
	}

	return results, nil
}

func GetForms(db *mongo.Database) ([]string, error) {
	pipeline := mongo.Pipeline{bson.D{{"$group", bson.D{{"_id", "$form"}}}}}
	cur, err := db.Collection("sb_forms").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var names []string
	for cur.Next(ctx) {
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
