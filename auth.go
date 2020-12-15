package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/gbrlsnchs/jwt/v3"
)

const (
	rootRole = 100
)

// Auth represents an authenticated user.
type Auth struct {
	AccountID primitive.ObjectID
	UserID    primitive.ObjectID
	Email     string
	Role      int
}

// JWTPayload contains the current user token
type JWTPayload struct {
	jwt.Payload
	Token string `json:"token,omitempty"`
}

var hs = jwt.NewHS256([]byte(os.Getenv("JWT_SECRET")))

var tokens map[string]Auth = make(map[string]Auth)

func auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Authorization")

		if len(key) == 0 {
			// if they requested a public repo we let them continue
			// to next security check.
			if strings.HasPrefix(r.URL.Path, "/db/pub_") || strings.HasPrefix(r.URL.Path, "/query/pub_") {
				a := Auth{
					AccountID: primitive.NewObjectID(),
					UserID:    primitive.NewObjectID(),
					Email:     "",
					Role:      0,
				}

				ctx := context.WithValue(r.Context(), ContextAuth, a)

				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			http.Error(w, "missing authorization HTTP header", http.StatusUnauthorized)
			return
		} else if strings.HasPrefix(key, "Bearer ") == false {
			http.Error(w,
				fmt.Sprintf("invalid authorization HTTP header, should be: Bearer your-token, but we got %s", key),
				http.StatusBadRequest,
			)
			return
		}

		key = strings.Replace(key, "Bearer ", "", -1)

		var pl JWTPayload
		if _, err := jwt.Verify([]byte(key), hs, &pl); err != nil {
			http.Error(w, fmt.Sprintf("could not verify your authentication token: %s", err.Error()), http.StatusBadRequest)
			return
		}

		ctx := r.Context()

		conf, ok := ctx.Value(ContextBase).(BaseConfig)
		if !ok {
			http.Error(w, "invalid StaticBackend public key", http.StatusUnauthorized)
			return
		}

		db := client.Database(conf.Name)

		a, ok := tokens[pl.Token]
		if ok {
			ctx = context.WithValue(ctx, ContextAuth, a)
		} else {
			parts := strings.Split(key, "|")
			if len(parts) != 2 {
				http.Error(w, "invalid authentication token", http.StatusUnauthorized)
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
				Role:      token.Role,
			}
			tokens[pl.Token] = a

			ctx = context.WithValue(ctx, ContextAuth, a)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requireRoot(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Authorization")

		if len(key) == 0 {
			http.Error(w, "missing authorization HTTP header", http.StatusUnauthorized)
			return
		} else if strings.HasPrefix(key, "Bearer ") == false {
			http.Error(w,
				fmt.Sprintf("invalid authorization HTTP header, should be: Bearer your-token, but we got %s", key),
				http.StatusBadRequest,
			)
			return
		}

		key = strings.Replace(key, "Bearer ", "", -1)

		parts := strings.Split(key, "|")
		if len(parts) != 3 {
			http.Error(w, "invalid root token", http.StatusBadRequest)
			return
		}

		id, err := primitive.ObjectIDFromHex(parts[0])
		if err != nil {
			http.Error(w, "invalid root token", http.StatusBadRequest)
			return
		}

		acctID, err := primitive.ObjectIDFromHex(parts[1])
		if err != nil {
			http.Error(w, "invalid root token", http.StatusBadRequest)
			return
		}

		token := parts[2]

		ctx := r.Context()
		conf, ok := ctx.Value(ContextBase).(BaseConfig)
		if !ok {
			http.Error(w, "invalid StaticBackend public key", http.StatusUnauthorized)
			return
		}

		db := client.Database(conf.Name)

		filter := bson.M{
			fieldID:        id,
			fieldAccountID: acctID,
			fieldToken:     token,
		}
		sr := db.Collection("sb_tokens").FindOne(context.Background(), filter)

		var tok Token
		if err := sr.Decode(&tok); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		} else if tok.Role < rootRole {
			http.Error(w, "not enough permission", http.StatusUnauthorized)
			return
		}

		a := Auth{
			AccountID: tok.AccountID,
			UserID:    tok.ID,
			Email:     tok.Email,
			Role:      tok.Role,
		}

		ctx = context.WithValue(ctx, ContextAuth, a)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
