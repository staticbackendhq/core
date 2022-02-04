package staticbackend

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/middleware"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func submitForm(w http.ResponseWriter, r *http.Request) {
	conf, _, err := middleware.Extract(r, false)
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
	conf, _, err := middleware.Extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	curDB := client.Database(conf.Name)

	formName := r.URL.Query().Get("name")

	results, err := internal.ListFormSubmissions(curDB, formName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, results)
}
