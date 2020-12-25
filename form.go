package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func submitForm(w http.ResponseWriter, r *http.Request) {
	conf, _, err := extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db := client.Database(conf.Name)

	form := ""

	_, r.URL.Path = ShiftPath(r.URL.Path)
	form, r.URL.Path = ShiftPath(r.URL.Path)

	doc := bson.M{
		"_id":       primitive.NewObjectID(),
		"form":      form,
		"sb_posted": time.Now(),
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// if there's something in the _hp_ field, it's a bot
	if len(r.Form.Get("_hp_")) > 0 {
		http.Error(w, "invalid form field present", http.StatusBadRequest)
		return
	}

	for key, val := range r.Form {
		k := strings.ToLower(key)
		if k != "_id" && k != "form" {
			doc[key] = strings.Join(val, " ; ")
		}
	}

	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	if _, err := db.Collection("sb_forms").InsertOne(ctx, doc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}

func listForm(w http.ResponseWriter, r *http.Request) {
	conf, _, err := extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db := client.Database(conf.Name)

	opt := options.Find()
	opt.SetLimit(100)
	opt.SetSort(bson.M{fieldID: -1})

	filter := bson.M{}
	if fn := r.URL.Query().Get("name"); len(fn) > 0 {
		filter["form"] = fn
	}

	cur, err := db.Collection("sb_forms").Find(context.Background(), filter, opt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cur.Close(context.Background())

	var results []bson.M

	for cur.Next(context.Background()) {
		var result bson.M
		err := cur.Decode(&result)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		result["id"] = result["_id"]
		delete(result, fieldID)

		results = append(results, result)
	}
	if err := cur.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(results) == 0 {
		results = make([]bson.M, 1)
	}

	respond(w, http.StatusOK, results)
}
