package mongo

import (
	"context"
	"time"

	"github.com/staticbackendhq/core/internal"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Mongo struct {
	Client *mongo.Client
	Ctx    context.Context
}

func New(client *mongo.Client) internal.Persister {
	return &Mongo{
		Client: client,
		Ctx:    context.Background(),
	}
}

func (mg *Mongo) Ping() error {
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	return mg.Client.Ping(ctx, readpref.Primary())
}
