package function

import (
	"encoding/json"
	"time"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/database"
	"github.com/staticbackendhq/core/email"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/model"
	"github.com/staticbackendhq/core/search"

	"github.com/go-co-op/gocron"
)

type TaskScheduler struct {
	Volatile  cache.Volatilizer
	DataStore database.Persister
	Search    *search.Search
	Email     email.Mailer
	Log       *logger.Logger

	Scheduler *gocron.Scheduler
}

func (ts *TaskScheduler) Start() {
	tasks, err := ts.DataStore.ListTasks()
	if err != nil {
		ts.Log.Error().Err(err).Msg("error loading tasks")
		return
	}
	ts.Scheduler = gocron.NewScheduler(time.UTC)
	ts.Scheduler.TagsUnique()

	for _, task := range tasks {
		_, err := ts.Scheduler.Cron(task.Interval).Tag(task.ID).Do(ts.run, task)
		if err != nil {
			ts.Log.Error().Err(err).Msgf("error scheduling this task: %s", task.ID)
		}
	}

	ts.Scheduler.StartBlocking()
}

func (ts *TaskScheduler) AddOnTheFly(task model.Task) {
	_, err := ts.Scheduler.Cron(task.Interval).Tag(task.ID).Do(ts.run, task)
	if err != nil {
		ts.Log.Error().Err(err).Msgf("error scheduling this task: %s", task.ID)
	}
}

func (ts *TaskScheduler) CancelTask(id string) error {
	return ts.Scheduler.RemoveByTag(id)
}

func (ts *TaskScheduler) run(task model.Task) {
	ts.Log.Info().Msgf("executing job:%s typed:%s value:%s", task.Name, task.Type, task.Value)

	// the task must run as the root base user
	var auth model.Auth
	if err := ts.Volatile.GetTyped("root:"+task.BaseName, &auth); err != nil {
		tok, err := ts.DataStore.GetRootForBase(task.BaseName)
		if err != nil {
			ts.Log.Error().Err(err).Msgf("error finding root token for base %s", task.BaseName)

			return
		}

		auth = model.Auth{
			AccountID: tok.AccountID,
			UserID:    tok.ID,
			Email:     tok.Email,
			Role:      tok.Role,
			Token:     tok.Token,
		}

		if err := ts.Volatile.SetTyped("root:"+task.BaseName, auth); err != nil {
			ts.Log.Error().Err(err).Msg("error setting auth inside TaskScheduler.run")
			return
		}
	}

	switch task.Type {
	case model.TaskTypeFunction:
		ts.execFunction(auth, task)
	case model.TaskTypeMessage:
		ts.sendMessage(auth, task)
	}
}

func (ts *TaskScheduler) execFunction(auth model.Auth, task model.Task) {
	fn, err := ts.DataStore.GetFunctionForExecution(task.BaseName, task.Value)
	if err != nil {
		ts.Log.Error().Err(err).Msgf("cannot find function %s on task %s", task.Value, task.ID)
		return
	}

	exe := &ExecutionEnvironment{
		Auth:      auth,
		BaseName:  task.BaseName,
		DataStore: ts.DataStore,
		Volatile:  ts.Volatile,
		Search:    ts.Search,
		Email:     ts.Email,
		Data:      fn,
		Log:       ts.Log,
	}

	var meta model.MetaMessage

	if len(task.Meta) > 0 {
		if err := json.Unmarshal([]byte(task.Meta), &meta); err != nil {
			ts.Log.Warn().Msgf("unable to get meta data for type MetaMessage for task: %s", task.ID)
			return
		}
	}

	msg := model.Command{
		Channel:       task.Name,
		Type:          model.MsgTypeFunctionCall,
		Data:          meta.Data,
		Auth:          auth,
		Base:          task.BaseName,
		IsSystemEvent: true,
	}

	if err := exe.Execute(msg); err != nil {
		ts.Log.Error().Err(err).Msgf("error executing function %s", task.Value)
	}
}

func (ts *TaskScheduler) sendMessage(auth model.Auth, task model.Task) {
	token := auth.ReconstructToken()

	var meta model.MetaMessage

	if len(task.Meta) > 0 {
		if err := json.Unmarshal([]byte(task.Meta), &meta); err != nil {
			ts.Log.Warn().Msgf("unable to get meta data for type MetaMessage for task: %s", task.ID)
			return
		}
	}

	msg := model.Command{
		SID:     task.ID,
		Type:    task.Value,
		Data:    meta.Data,
		Channel: meta.Channel,
		Token:   token,
		Auth:    auth,
		Base:    task.BaseName,
	}

	if err := ts.Volatile.Publish(msg); err != nil {
		ts.Log.Error().Err(err).Msgf("error publishing message from task: %s", task.ID)
	}
}
