package middleware

import (
	"context"
	"log"
	"net/http"
	"staticbackend/internal"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func WithDB(client *mongo.Client) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("SB-PUBLIC-KEY")

			// we check in query string (used for SSE)
			if len(key) == 0 {
				key = r.URL.Query().Get("sbpk")
			}

			if len(key) == 0 {
				http.Error(w, "invalid StaticBackend public key", http.StatusUnauthorized)
				log.Println("invalid StaticBackend key")
				return
			}

			ctx := r.Context()

			conf, ok := internal.Bases[key]
			if ok {
				ctx = context.WithValue(ctx, ContextBase, conf)
			} else {
				// let's try to see if they are allow to use a database
				db := client.Database("sbsys")

				oid, err := primitive.ObjectIDFromHex(key)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					log.Println("unable to convert id to ObjectID", err)
					return
				}

				dbCtx, _ := context.WithTimeout(context.Background(), 2*time.Second)
				sr := db.Collection("bases").FindOne(dbCtx, bson.M{"_id": oid})
				if err := sr.Decode(&conf); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					log.Println("cannot find this base", err, " Key:", oid)
					return
				}

				ctx = context.WithValue(ctx, ContextBase, conf)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
