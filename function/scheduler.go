package function

import (
	"time"

	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/logger"

	"github.com/go-co-op/gocron"
	"go.mongodb.org/mongo-driver/mongo"
)

type TaskScheduler struct {
	Client    *mongo.Client
	Volatile  internal.PubSuber
	DataStore internal.Persister
	Scheduler *gocron.Scheduler
	log       *logger.Logger
}

func (ts *TaskScheduler) Start() {
	tasks, err := ts.DataStore.ListTasks()
	if err != nil {
		ts.log.Error().Err(err).Msg("error loading tasks")
		return
	}

	ts.Scheduler = gocron.NewScheduler(time.UTC)
	ts.Scheduler.TagsUnique()

	for _, task := range tasks {
		_, err := ts.Scheduler.Cron(task.Interval).Tag(task.ID).Do(ts.run, task)
		if err != nil {
			ts.log.Error().Err(err).Msgf("error scheduling this task: %s", task.ID)
		}
	}
}

func (ts *TaskScheduler) run(task internal.Task) {
	// the task must run as the root base user
	var auth internal.Auth
	if err := ts.Volatile.GetTyped("root:"+task.BaseName, &auth); err != nil {
		tok, err := ts.DataStore.GetRootForBase(task.BaseName)
		if err != nil {
			ts.log.Error().Err(err).Msgf("error finding root token for base %s", task.BaseName)

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
			ts.log.Error().Err(err).Msg("error setting auth inside TaskScheduler.run")
			return
		}
	}

	switch task.Type {
	case internal.TaskTypeFunction:
		ts.execFunction(auth, task)
	case internal.TaskTypeMessage:
		ts.sendMessage(auth, task)
	}
}

func (ts *TaskScheduler) execFunction(auth internal.Auth, task internal.Task) {
	fn, err := ts.DataStore.GetFunctionForExecution(task.BaseName, task.Value)
	if err != nil {
		ts.log.Error().Err(err).Msgf("cannot find function %s on task %s", task.Value)
		return
	}

	exe := &ExecutionEnvironment{
		Auth:      auth,
		BaseName:  task.BaseName,
		DataStore: ts.DataStore,
		Volatile:  ts.Volatile,
		Data:      fn,
		log:       ts.log,
	}

	if err := exe.Execute(task.Name); err != nil {
		ts.log.Error().Err(err).Msgf("error executing function %s", task.Value)
	}
}

func (ts *TaskScheduler) sendMessage(auth internal.Auth, task internal.Task) {
	token := auth.ReconstructToken()

	meta, ok := task.Meta.(internal.MetaMessage)
	if !ok {
		ts.log.Warn().Msgf("unable to get meta data for type MetaMessage for task: %d", task.ID)
		return
	}

	msg := internal.Command{
		SID:     task.ID,
		Type:    task.Value,
		Data:    meta.Data,
		Channel: meta.Channel,
		Token:   token,
	}

	if err := ts.Volatile.Publish(msg); err != nil {
		ts.log.Error().Err(err).Msgf("error publishing message from task: %d", task.ID)
	}
}
