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
	Meta     string             `bson:"meta" json:"meta"`
	Interval string             `bson:"invertal" json:"interval"`
	LastRun  time.Time          `bson:"last" json:"last"`

	BaseName string `bson:"-" json:"base"`
}

func toLocalTask(t model.Task) LocalTask {
	id, err := primitive.ObjectIDFromHex(t.ID)
	if err != nil {
	}

	return LocalTask{
		ID:       id,
		Name:     t.Name,
		Type:     t.Type,
		Value:    t.Value,
		Meta:     t.Meta,
		Interval: t.Interval,
		LastRun:  t.LastRun,
	}
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

type LocalMetaMessage struct {
	Data    string `bson:"data" json:"data"`
	Channel string `bson:"channel" json:"channel"`
}

func (mg *Mongo) ListTasks() ([]model.Task, error) {
	bases, err := mg.ListDatabases()
	if err != nil {
		return nil, err
	}

	//TODO: Might be worth doing this concurrently
	var results []model.Task

	for _, base := range bases {
		tasks, err := mg.ListTasksByBase(base.Name)
		if err != nil {
			return nil, err
		}

		results = append(results, tasks...)
	}

	return results, nil
}

func (mg *Mongo) ListTasksByBase(dbName string) ([]model.Task, error) {
	db := mg.Client.Database(dbName)

	cur, err := db.Collection("sb_tasks").Find(mg.Ctx, bson.M{})
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

		t.BaseName = dbName

		tasks = append(tasks, fromLocalTask(t))
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (mg *Mongo) AddTask(dbName string, task model.Task) (string, error) {
	db := mg.Client.Database(dbName)

	task.ID = mg.NewID()
	v := toLocalTask(task)
	if _, err := db.Collection("sb_tasks").InsertOne(mg.Ctx, v); err != nil {
		return "", err
	}

	return task.ID, nil
}

func (mg *Mongo) DeleteTask(dbName, id string) error {
	db := mg.Client.Database(dbName)

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	filter := bson.M{FieldID: oid}
	if _, err := db.Collection("sb_tasks").DeleteOne(mg.Ctx, filter); err != nil {
		return err
	}
	return nil
}
