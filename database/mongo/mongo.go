package mongo

import (
	"context"
	"time"

	"github.com/staticbackendhq/core/internal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Mongo struct {
	Client          *mongo.Client
	Ctx             context.Context
	PublishDocument internal.PublishDocumentEvent
}

func New(client *mongo.Client, pubdoc internal.PublishDocumentEvent) internal.Persister {
	return &Mongo{
		Client:          client,
		Ctx:             context.Background(),
		PublishDocument: pubdoc,
	}
}

func (mg *Mongo) Ping() error {
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	return mg.Client.Ping(ctx, readpref.Primary())
}

func (mg *Mongo) CreateIndex(dbName, col, field string) error {
	db := mg.Client.Database(dbName)

	idx := mongo.IndexModel{
		Keys: bson.M{field: 1},
	}

	dbCol := db.Collection(internal.CleanCollectionName(col))

	if _, err := dbCol.Indexes().CreateOne(mg.Ctx, idx); err != nil {
		return err
	}
	return nil
}
