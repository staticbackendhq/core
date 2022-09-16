package mongo

import (
	"time"

	"github.com/staticbackendhq/core/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LocalTask struct {
	ID       primitive.ObjectID `bson:"_id" json:"id"`
	Name     string             `bson:"name" json:"name"`
	Type     string             `bson:"type" json:"type"`
	Value    string             `bson:"value" json:"value"`
	Meta     interface{}        `bson:"meta" json:"meta"`
	Interval string             `bson:"invertal" json:"interval"`
	LastRun  time.Time          `bson:"last" json:"last"`

	BaseName string `bson:"-" json:"base"`
}

type LocalMetaMessage struct {
	Data    string `bson:"data" json:"data"`
	Channel string `bson:"channel" json:"channel"`
}

func (mg *Mongo) ListTasks() ([]model.Task, error) {
	bases, err := mg.ListDatabases()
	if err != nil {
		return nil, err
	}

	filter := bson.M{}

	//TODO: Might be worth doing this concurrently
	var results []model.Task

	for _, base := range bases {
		db := mg.Client.Database(base.Name)
		cur, err := db.Collection("sb_tasks").Find(mg.Ctx, filter)
		if err != nil {
			return nil, err
		}
		defer cur.Close(mg.Ctx)

		var tasks []model.Task

		for cur.Next(mg.Ctx) {
			var t LocalTask
			if err := cur.Decode(&t); err != nil {
				return nil, err
			}

			t.BaseName = base.Name

			tasks = append(tasks, fromLocalTask(t))
		}
		if err := cur.Err(); err != nil {
			return nil, err
		}

		results = append(results, tasks...)
	}

	return results, nil
}

func fromLocalTask(lt LocalTask) model.Task {
	return model.Task{
		ID:       lt.ID.Hex(),
		Name:     lt.Name,
		Type:     lt.Type,
		Value:    lt.Value,
		Meta:     lt.Meta,
		Interval: lt.Interval,
		LastRun:  lt.LastRun,
		BaseName: lt.BaseName,
	}
}
