package main

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config, auth, err := extract(r, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db := client.Database(config.Name)

	file, h, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// check for file size
	// TODO: This should be based on current plan
	if h.Size/(1000*1000) > 64 {
		http.Error(w, "file size exeeded your limit", http.StatusBadRequest)
		return
	}

	ext := filepath.Ext(h.Filename)

	//TODO: Remove all but a-zA-Z/ from name

	name := r.Form.Get("name")
	if len(name) == 0 {
		name = primitive.NewObjectID().Hex()
	}

	sess, err := session.NewSession(&aws.Config{Region: aws.String("ca-central-1")})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fileKey := fmt.Sprintf("%s/%s/%s%s",
		auth.AccountID.Hex(),
		config.Name,
		name,
		ext,
	)

	svc := s3.New(sess)
	obj := &s3.PutObjectInput{}
	obj.Body = file
	obj.ACL = aws.String(s3.ObjectCannedACLPublicRead)
	obj.Bucket = aws.String("files.staticbackend.com")
	obj.Key = aws.String(fileKey)

	if _, err := svc.PutObject(obj); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	url := fmt.Sprintf(
		"https://cdn.staticbackend.com/%s",
		fileKey,
	)

	doc := bson.M{
		"accountId": auth.AccountID,
		"key":       fileKey,
		"url":       url,
		"size":      h.Size,
		"on":        time.Now(),
	}

	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	res, err := db.Collection("sb_files").InsertOne(ctx, doc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newID, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		http.Error(w, "unable to retrived the inserted id", http.StatusInternalServerError)
		return
	}

	data := new(struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	})
	data.ID = newID.Hex()
	data.URL = url

	respond(w, http.StatusOK, data)
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
	config, _, err := extract(r, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db := client.Database(config.Name)

	oid, err := primitive.ObjectIDFromHex(r.URL.Query().Get("id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	var result bson.M

	filter := bson.M{fieldID: oid}

	sr := db.Collection("sb_files").FindOne(ctx, filter)
	if err := sr.Decode(&result); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if err := sr.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fileKey, ok := result["key"].(string)
	if !ok {
		http.Error(w, "unable to retrive the file id", http.StatusInternalServerError)
		return
	}

	sess, err := session.NewSession(&aws.Config{Region: aws.String("ca-central-1")})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	svc := s3.New(sess)
	obj := &s3.DeleteObjectInput{
		Bucket: aws.String("files.staticbackend.com"),
		Key:    aws.String(fileKey),
	}
	if _, err := svc.DeleteObject(obj); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := db.Collection("sb_files").DeleteOne(ctx, filter); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, true)
}
