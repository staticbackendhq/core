package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type Account struct {
	ID    primitive.ObjectID `bson:"_id" json:"id"`
	Email string             `bson:"email" json:"email"`
}

type Token struct {
	ID        primitive.ObjectID `bson:"_id" json:"id"`
	AccountID primitive.ObjectID `bson:"accountId" json:"accountId"`
	Token     string             `bson:"token" json:"token"`
	Email     string             `bson:"email" json:"email"`
	Password  string             `bson:"pw" json:"-"`
}

type Login struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func login(w http.ResponseWriter, r *http.Request) {
	conf, ok := r.Context().Value(ContextBase).(BaseConfig)
	if !ok {
		http.Error(w, "invalid StaticBackend key", http.StatusUnauthorized)
		return
	}

	db := client.Database(conf.Name)

	var l Login
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	sr := db.Collection("sb_tokens").FindOne(ctx, bson.M{"email": l.Email})

	var tok Token
	if err := sr.Decode(&tok); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(tok.Password), []byte(l.Password)); err != nil {
		http.Error(w, "invalid email/password", http.StatusNotFound)
		return
	}

	tokens[fmt.Sprintf("%s|%s", tok.ID.Hex(), tok.Token)] = Auth{
		AccountID: tok.AccountID,
		UserID:    tok.ID,
		Email:     tok.Email,
	}

	respond(w, http.StatusOK, tok)
}

func register(w http.ResponseWriter, r *http.Request) {
	conf, ok := r.Context().Value(ContextBase).(BaseConfig)
	if !ok {
		http.Error(w, "invalid StaticBackend key", http.StatusUnauthorized)
		log.Println("invalid StaticBackend key")
		return
	}

	db := client.Database(conf.Name)

	var l Login
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	a := Account{
		ID:    primitive.NewObjectID(),
		Email: l.Email,
	}

	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	_, err := db.Collection("sb_accounts").InsertOne(ctx, a)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b, err := bcrypt.GenerateFromPassword([]byte(l.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tok := Token{
		ID:        primitive.NewObjectID(),
		AccountID: a.ID,
		Email:     l.Email,
		Token:     primitive.NewObjectID().Hex(),
		Password:  string(b),
	}

	_, err = db.Collection("sb_tokens").InsertOne(ctx, tok)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tokens[fmt.Sprintf("%s|%s", tok.ID.Hex(), tok.Token)] = Auth{
		AccountID: tok.AccountID,
		UserID:    tok.ID,
		Email:     tok.Email,
	}

	respond(w, http.StatusOK, tok)
}
