package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Auth represents an authenticated user.
type Auth struct {
	AccountID primitive.ObjectID
	UserID    primitive.ObjectID
	Email     string
}

var tokens map[string]Auth = make(map[string]Auth)

func auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-API-KEY")

		if len(key) == 0 {
			http.Error(w, "invalid authentication", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()

		conf, ok := ctx.Value(ContextBase).(BaseConfig)
		if !ok {
			http.Error(w, "invalid StaticBackend key", http.StatusUnauthorized)
			return
		}

		db := client.Database(conf.Name)

		a, ok := tokens[key]
		if ok {
			ctx = context.WithValue(ctx, ContextAuth, a)
		} else {
			parts := strings.Split(key, "|")
			if len(parts) != 2 {
				http.Error(w, "invalid API key format", http.StatusUnauthorized)
				return
			}

			id, err := primitive.ObjectIDFromHex(parts[0])
			if err != nil {
				http.Error(w, "invalid API key format", http.StatusUnauthorized)
				return
			}

			ctxAuth, _ := context.WithTimeout(context.Background(), 2*time.Second)
			sr := db.Collection("sb_tokens").FindOne(ctxAuth, bson.M{"_id": id, "token": parts[1]})

			var token Token
			if err := sr.Decode(&token); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			a := Auth{
				AccountID: token.AccountID,
				UserID:    token.ID,
				Email:     token.Email,
			}
			tokens[key] = a

			ctx = context.WithValue(ctx, ContextAuth, a)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
