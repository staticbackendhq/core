package function

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/database"
	"github.com/staticbackendhq/core/email"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/model"
	"github.com/staticbackendhq/core/search"

	"github.com/go-co-op/gocron/v2"
)

type TaskScheduler struct {
	Volatile  cache.Volatilizer
	DataStore database.Persister
	Search    *search.Search
	Email     email.Mailer
	Log       *logger.Logger

	Scheduler gocron.Scheduler

	mu      sync.Mutex
	stop    chan struct{}
	done    chan struct{}
	running bool
	stopped bool
}

type taskAuthCache struct {
	AccountID string `json:"accountId"`
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	Role      int    `json:"role"`
	Token     string `json:"token"`
}

func (t taskAuthCache) auth() model.Auth {
	return model.Auth{
		AccountID: t.AccountID,
		UserID:    t.UserID,
		Email:     t.Email,
		Role:      t.Role,
		Token:     t.Token,
	}
}

func (ts *TaskScheduler) Start() {
	ts.ensureLifecycle()
	ts.mu.Lock()
	if ts.stopped {
		done := ts.done
		ts.mu.Unlock()
		close(done)
		return
	}
	ts.running = true
	done := ts.done
	stop := ts.stop
	ts.mu.Unlock()

	defer func() {
		ts.mu.Lock()
		ts.running = false
		ts.mu.Unlock()
		close(done)
	}()

	select {
	case <-stop:
		return
	default:
	}

	tasks, err := ts.DataStore.ListTasks()
	if err != nil {
		ts.Log.Error().Err(err).Msg("error loading tasks")
		return
	}

	select {
	case <-stop:
		return
	default:
	}

	ts.ensureScheduler()
	if ts.Scheduler == nil {
		return
	}

	for _, task := range tasks {
		ts.addTask(task)
	}

	ts.Scheduler.Start()
	<-stop
}

// Stop stops the task scheduler and waits for Start to return.
func (ts *TaskScheduler) Stop(ctx context.Context) error {
	ts.ensureLifecycle()

	ts.mu.Lock()
	if !ts.stopped {
		close(ts.stop)
		ts.stopped = true
	}
	done := ts.done
	running := ts.running
	ts.mu.Unlock()

	if running {
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	ts.mu.Lock()
	scheduler := ts.Scheduler
	ts.mu.Unlock()
	if scheduler == nil {
		return nil
	}
	return scheduler.Shutdown()
}

func (ts *TaskScheduler) AddOnTheFly(task model.Task) {
	ts.ensureScheduler()
	ts.addTask(task)
}

func (ts *TaskScheduler) addTask(task model.Task) {
	if ts.Scheduler == nil {
		ts.Log.Error().Msgf("scheduler is not initialized; cannot schedule task: %s", task.ID)
		return
	}

	_, err := ts.Scheduler.NewJob(
		gocron.CronJob(task.Interval, false),
		gocron.NewTask(ts.run, task),
		gocron.WithTags(task.ID),
	)
	if err != nil {
		ts.Log.Error().Err(err).Msgf("error scheduling this task: %s", task.ID)
	}
}

func (ts *TaskScheduler) CancelTask(id string) error {
	ts.ensureScheduler()
	if ts.Scheduler == nil {
		return fmt.Errorf("scheduler is not initialized")
	}

	ts.Scheduler.RemoveByTags(id)
	return nil
}

func (ts *TaskScheduler) ensureScheduler() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.Scheduler != nil {
		return
	}
	if ts.stopped {
		return
	}

	scheduler, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))
	if err != nil {
		ts.Log.Error().Err(err).Msg("error creating task scheduler")
		return
	}

	ts.Scheduler = scheduler
}

func (ts *TaskScheduler) ensureLifecycle() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.stop == nil {
		ts.stop = make(chan struct{})
	}
	if ts.done == nil {
		ts.done = make(chan struct{})
	}
}

func (ts *TaskScheduler) run(task model.Task) {
	ts.markTaskRan(task)

	// the task must run as the root base user
	var auth model.Auth
	var cachedAuth taskAuthCache
	if err := ts.Volatile.GetTyped("root:"+task.BaseName, &cachedAuth); err == nil {
		auth = cachedAuth.auth()
	} else {
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

		if err := ts.Volatile.SetTyped("root:"+task.BaseName, taskAuthCache{
			AccountID: auth.AccountID,
			UserID:    auth.UserID,
			Email:     auth.Email,
			Role:      auth.Role,
			Token:     auth.Token,
		}); err != nil {
			ts.Log.Error().Err(err).Msg("error setting auth inside TaskScheduler.run")
			return
		}
	}

	switch task.Type {
	case model.TaskTypeFunction:
		ts.execFunction(auth, task)
	case model.TaskTypeMessage:
		ts.sendMessage(auth, task)
	case model.TaskTypeHTTP:
		ts.httpRequest(auth, task)
	}
}

func (ts *TaskScheduler) markTaskRan(task model.Task) {
	stored, err := ts.DataStore.GetTask(task.BaseName, task.ID)
	if err != nil {
		ts.Log.Error().Err(err).Msgf("error loading task before updating last run: %s", task.ID)
		return
	}

	stored.LastRun = time.Now().UTC()
	if err := ts.DataStore.UpdateTask(task.BaseName, stored); err != nil {
		ts.Log.Error().Err(err).Msgf("error updating last run for task: %s", task.ID)
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

func (ts *TaskScheduler) httpRequest(auth model.Auth, task model.Task) {
	token := auth.ReconstructToken()

	var meta model.MetaMessage
	headers := make(map[string]string)

	if len(task.Meta) > 0 {
		if err := json.Unmarshal([]byte(task.Meta), &meta); err != nil {
			ts.Log.Warn().Msgf("unable to get meta data for type MetaMessage for task: %s", task.ID)
			return
		}

		if err := json.Unmarshal([]byte(meta.HTTPHeaders), &headers); err != nil {
			ts.Log.Err(err).Msg("unable to parse HTTP headers from meta data")
			return
		}
	}

	if len(meta.ContentType) == 0 {
		meta.ContentType = "application/x-www-form-urlencoded"
	}

	if len(meta.HTTPMethod) == 0 {
		meta.HTTPMethod = "POST"
	}

	body := ""
	if meta.ContentType == "application/json" {
		body = meta.Data
	} else {
		var v map[string]any
		if err := json.Unmarshal([]byte(meta.Data), &v); err != nil {
			ts.Log.Warn().Err(err).Msg("unable to parse meta data")
			return
		}

		data := url.Values{}
		for key, val := range v {
			data.Add(key, fmt.Sprintf("%v", val))
		}

		body = data.Encode()
	}

	req, err := http.NewRequest(meta.HTTPMethod, task.Value, strings.NewReader(body))
	if err != nil {
		ts.Log.Err(err).Msg("unable to construct the HTTP request")
		return
	}

	req.Header.Add("Content-Type", meta.ContentType)

	for key, val := range headers {
		req.Header.Add(key, val)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ts.Log.Err(err).Msg("error executing HTTP request")
		return
	}
	defer func() { _ = resp.Body.Close() }()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		ts.Log.Err(err).Msg("unable to read HTTP response body")
		return
	}

	msg := model.Command{
		SID:     "system",
		Type:    model.MsgTypeHTTPResponse,
		Channel: fmt.Sprintf(`%s-http-response`, task.Name),
		Data:    string(b),
		Token:   token,
		Auth:    auth,
		Base:    task.BaseName,
	}

	if err := ts.Volatile.Publish(msg); err != nil {
		ts.Log.Error().Err(err).Msgf("error publishing message from task: %s", task.ID)
	}
}
