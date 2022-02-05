package function

import (
	"log"
	"time"

	"github.com/staticbackendhq/core/internal"

	"github.com/go-co-op/gocron"
	"go.mongodb.org/mongo-driver/mongo"
)

type TaskScheduler struct {
	Client    *mongo.Client
	Volatile  internal.PubSuber
	DataStore internal.Persister
	Scheduler *gocron.Scheduler
}

func (ts *TaskScheduler) Start() {
	tasks, err := ts.DataStore.ListTasks()
	if err != nil {
		log.Println("error loading tasks: ", err)
		return
	}

	ts.Scheduler = gocron.NewScheduler(time.UTC)
	ts.Scheduler.TagsUnique()

	for _, task := range tasks {
		_, err := ts.Scheduler.Cron(task.Interval).Tag(task.ID.Hex()).Do(ts.run, task)
		if err != nil {
			log.Printf("error scheduling this task: %s -> %v\n", task.ID.Hex(), err)
		}
	}
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
			log.Println("error setting auth inside TaskScheduler.run: ", err)
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
