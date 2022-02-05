package mongo

import (
	"context"

	"github.com/staticbackendhq/core/internal"
	"go.mongodb.org/mongo-driver/mongo"
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
