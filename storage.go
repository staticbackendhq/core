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

	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fileKey := fmt.Sprintf("%s/%s/%s.%s",
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
	if _, err := db.Collection("sb_files").InsertOne(ctx, doc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, http.StatusOK, url)
}
