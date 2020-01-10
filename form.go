package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
