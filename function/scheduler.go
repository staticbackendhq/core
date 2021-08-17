package function

import (
	"context"
	"log"
	"staticbackend/db"
	"staticbackend/internal"
	"time"

	"github.com/go-co-op/gocron"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type TaskScheduler struct {
	Client    *mongo.Client
	Volatile  internal.PubSuber
	Scheduler *gocron.Scheduler
}

const (
	TaskTypeFunction = "function"
	TaskTypeMessage  = "message"
)

type Task struct {
	ID       primitive.ObjectID `bson:"_id" json:"id"`
	Name     string             `bson:"name" json:"name"`
	Type     string             `bson:"type" json:"type"`
	Value    string             `bson:"value" json:"value"`
	Meta     interface{}        `bson:"meta" json:"meta"`
	Interval string             `bson:"invertal" json:"interval"`
	LastRun  time.Time          `bson:"last" json:"last"`

	BaseName string `bson:"-" json:"base"`
}

type MetaMessage struct {
	Data    string `bson:"data" json:"data"`
	Channel string `bson:"channel" json:"channel"`
}

func (ts *TaskScheduler) Start() {
	tasks, err := ts.listTasks()
	if err != nil {
		log.Println("error loading tasks: %v", err)
		return
	}

	ts.Scheduler = gocron.NewScheduler(time.UTC)
	ts.Scheduler.TagsUnique()

	for _, task := range tasks {
		_, err := ts.Scheduler.Cron(task.Interval).Tag(task.ID.Hex()).Do(ts.run, task)
		if err != nil {
			log.Println("error scheduling this task: %s -> %v", task.ID.Hex(), err)
		}
	}
}

func (ts *TaskScheduler) listTasks() ([]Task, error) {
	bases, err := internal.ListDatabases(ts.Client.Database("sbsys"))
	if err != nil {
		return nil, err
	}

	filter := bson.M{}

	//TODO: Might be worth doing this concurrently
	var results []Task

	ctx := context.Background()

	for _, base := range bases {
		db := ts.Client.Database(base.Name)
		cur, err := db.Collection("sb_tasks").Find(ctx, filter)
		if err != nil {
			return nil, err
		}
		defer cur.Close(ctx)

		var tasks []Task

		for cur.Next(ctx) {
			var t Task
			if err := cur.Decode(&t); err != nil {
				return nil, err
			}

			t.BaseName = base.Name

			tasks = append(tasks, t)
		}
		if err := cur.Err(); err != nil {
			return nil, err
		}

		results = append(results, tasks...)
	}

	return results, nil
}

func (ts *TaskScheduler) run(task Task) {
	curDB := ts.Client.Database(task.BaseName)

	// the task must run as the root base user
	var auth internal.Auth
	if err := ts.Volatile.GetTyped("root:"+task.BaseName, &auth); err != nil {
		tok, err := internal.GetRootForBase(curDB)
		if err != nil {
			log.Printf("error finding root token for base %s: %v\n", task.BaseName, err)
			return
		}

		auth = internal.Auth{
			AccountID: tok.AccountID,
			UserID:    tok.ID,
			Email:     tok.Email,
			Role:      tok.Role,
			Token:     tok.Token,
		}

		if err := ts.Volatile.SetTyped("root:"+task.BaseName, auth); err != nil {
			log.Printf("error setting auth inside TaskScheduler.run: ", err)
			return
		}
	}

	switch task.Type {
	case TaskTypeFunction:
		ts.execFunction(curDB, auth, task)
	case TaskTypeMessage:
		ts.sendMessage(curDB, auth, task)
	}
}

func (ts *TaskScheduler) execFunction(curDB *mongo.Database, auth internal.Auth, task Task) {

	fn, err := GetForExecution(curDB, task.Value)
	if err != nil {
		log.Printf("cannot find function %s on task %s", task.Value, task.ID.Hex())
		return
	}

	exe := &ExecutionEnvironment{
		Auth:     auth,
		DB:       curDB,
		Base:     &db.Base{PublishDocument: ts.Volatile.PublishDocument},
		Volatile: ts.Volatile,
		Data:     fn,
	}

	if err := exe.Execute(task.Name); err != nil {
		log.Printf("error executing function %s: %v", task.Value, err)
	}
}

func (ts *TaskScheduler) sendMessage(curDB *mongo.Database, auth internal.Auth, task Task) {
	token := auth.ReconstructToken()

	meta, ok := task.Meta.(MetaMessage)
	if !ok {
		log.Println("unable to get meta data for type MetaMessage for task: ", task.ID.Hex())
		return
	}

	msg := internal.Command{
		SID:     task.ID.Hex(),
		Type:    task.Value,
		Data:    meta.Data,
		Channel: meta.Channel,
		Token:   token,
	}

	if err := ts.Volatile.Publish(msg); err != nil {
		log.Println("error publishing message from task", task.ID.Hex(), err)
	}
}
