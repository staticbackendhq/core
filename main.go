package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	client *mongo.Client
)

func main() {
	dbHost := flag.String("host", "localhost", "Hostname for mongodb")
	port := flag.String("port", "8099", "HTTP port to listen on")
	flag.Parse()

	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	cl, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://"+*dbHost+":27017"))
	if err != nil {
		log.Fatal("cannot connect to mongo: ", err)
	}

	if err := cl.Ping(ctx, readpref.Primary()); err != nil {
		log.Fatal("Ping failed: ", err)
		return
	}

	client = cl

	http.Handle("/login", chain(http.HandlerFunc(login), withDB, cors))
	http.Handle("/register", chain(http.HandlerFunc(register), withDB, cors))

	// database routes
	http.Handle("/add/", chain(http.HandlerFunc(add), auth, withDB, cors))
	http.Handle("/list/", chain(http.HandlerFunc(list), auth, withDB, cors))
	http.Handle("/get/", chain(http.HandlerFunc(get), auth, withDB, cors))
	http.Handle("/query/", chain(http.HandlerFunc(query), auth, withDB, cors))
	http.Handle("/update/", chain(http.HandlerFunc(update), auth, withDB, cors))
	http.Handle("/delete/", chain(http.HandlerFunc(del), auth, withDB, cors))
	http.Handle("/newid", chain(http.HandlerFunc(newID), auth, withDB, cors))

	// forms routes
	http.Handle("/postform/", chain(http.HandlerFunc(submitForm), withDB, cors))

	// storage
	http.Handle("/storage/upload", chain(http.HandlerFunc(upload), auth, withDB, cors))

	http.HandleFunc("/ping", ping)

	log.Fatal(http.ListenAndServe(":"+*port, nil))
}

type chainer func(h http.Handler) http.Handler

func chain(h http.Handler, middlewares ...chainer) http.Handler {
	next := h
	for _, m := range middlewares {
		next = m(next)
	}
	return next
}

func ping(w http.ResponseWriter, r *http.Request) {
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		http.Error(w, "connection failed to database, I'm down.", http.StatusInternalServerError)
		return
	}
	respond(w, http.StatusOK, true)
}
