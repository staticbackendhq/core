package function

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/database"
	"github.com/staticbackendhq/core/database/memory"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/model"
)

func TestTaskSchedulerUsesRootAuthForFunctionTask(t *testing.T) {
	baseName := fmt.Sprintf("sched_%d", time.Now().UnixNano())
	ds, rootAuth := newSchedulerTestStore(t, baseName)
	vol := cache.NewDevCache()

	fn := model.ExecData{
		FunctionName: "scheduled-create",
		TriggerTopic: "schedule",
		Code: `function handle(channel, type, data) {
			var result = create("scheduled_runs", { name: data.name });
			if (!result.ok) {
				log("ERROR: " + result.content);
			}
		}`,
		Version: 1,
	}
	fnID, err := ds.AddFunction(baseName, fn)
	if err != nil {
		t.Fatal(err)
	}
	fn.ID = fnID

	ts := &TaskScheduler{
		Volatile:  vol,
		DataStore: ds,
	}
	task := model.Task{
		ID:       "task-root-auth",
		Name:     "daily-root-auth",
		Type:     model.TaskTypeFunction,
		Value:    fn.FunctionName,
		Meta:     `{"data":"{\"name\":\"from schedule\"}"}`,
		Interval: "0 1 * * *",
		BaseName: baseName,
	}

	ts.run(task)

	var cached taskAuthCache
	if err := vol.GetTyped("root:"+baseName, &cached); err != nil {
		t.Fatal(err)
	}
	if cached.AccountID != rootAuth.AccountID || cached.UserID != rootAuth.UserID || cached.Token != rootAuth.Token {
		t.Fatalf("scheduler cached wrong root auth: got %+v want %+v", cached, rootAuth)
	}

	result, err := ds.ListDocuments(rootAuth, baseName, "scheduled_runs", model.ListParams{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 {
		t.Fatalf("expected one scheduled run document, got %d", result.Total)
	}
	doc := result.Results[0]
	if doc["accountId"] != rootAuth.AccountID {
		t.Fatalf("expected document accountId %q got %v", rootAuth.AccountID, doc["accountId"])
	}
	if doc["sb_ownerId"] != rootAuth.UserID {
		t.Fatalf("expected document sb_ownerId %q got %v", rootAuth.UserID, doc["sb_ownerId"])
	}

	waitForFunctionHistory(t, ds, baseName, fn.FunctionName)

	msgs := make(chan model.Command, 1)
	done := make(chan bool)
	defer close(done)
	go vol.Subscribe(msgs, rootAuth.ReconstructToken(), "scheduled-events", done)
	time.Sleep(10 * time.Millisecond)

	ts.run(model.Task{
		ID:       "task-root-message",
		Name:     "root-message",
		Type:     model.TaskTypeMessage,
		Value:    "scheduled.event",
		Meta:     `{"data":"{}","channel":"scheduled-events"}`,
		Interval: "0 1 * * *",
		BaseName: baseName,
	})

	select {
	case msg := <-msgs:
		if msg.Token != rootAuth.ReconstructToken() {
			t.Fatalf("expected published token %q got %q", rootAuth.ReconstructToken(), msg.Token)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for scheduled message")
	}
}

func TestTaskSchedulerFunctionTaskWithoutMetaUsesEmptyData(t *testing.T) {
	baseName := fmt.Sprintf("sched_empty_meta_%d", time.Now().UnixNano())
	ds, rootAuth := newSchedulerTestStore(t, baseName)
	vol := cache.NewDevCache()

	fn := model.ExecData{
		FunctionName: "empty-meta",
		TriggerTopic: "schedule",
		Code: `function handle(channel, type, data) {
			if (channel !== "empty-meta-task") {
				throw new Error("unexpected channel: " + channel);
			}
			if (type !== "fn_call") {
				throw new Error("unexpected type: " + type);
			}
			if (!data || Object.keys(data).length !== 0) {
				throw new Error("expected empty data object");
			}

			var result = create("scheduled_empty_meta_runs", { ok: true });
			if (!result.ok) {
				throw new Error(result.content);
			}
		}`,
		Version: 1,
	}
	if _, err := ds.AddFunction(baseName, fn); err != nil {
		t.Fatal(err)
	}

	task := model.Task{
		Name:     "empty-meta-task",
		Type:     model.TaskTypeFunction,
		Value:    fn.FunctionName,
		Interval: "0 1 * * *",
		BaseName: baseName,
	}
	taskID, err := ds.AddTask(baseName, task)
	if err != nil {
		t.Fatal(err)
	}
	task.ID = taskID

	ts := &TaskScheduler{
		Volatile:  vol,
		DataStore: ds,
	}
	ts.run(task)

	waitForFunctionHistory(t, ds, baseName, fn.FunctionName)

	storedTask, err := ds.GetTask(baseName, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedTask.LastRun.IsZero() {
		t.Fatal("expected scheduled task last run to be recorded")
	}

	result, err := ds.ListDocuments(rootAuth, baseName, "scheduled_empty_meta_runs", model.ListParams{Page: 1, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 1 {
		t.Fatalf("expected one scheduled empty meta run document, got %d", result.Total)
	}
}

func TestTaskSchedulerAddAndCancelOnTheFly(t *testing.T) {
	ts := &TaskScheduler{Log: logger.Get(config.LoadConfig())}
	task := model.Task{
		ID:       "task-on-the-fly",
		Name:     "on-the-fly",
		Type:     model.TaskTypeMessage,
		Value:    "noop",
		Interval: "0 1 * * *",
	}

	ts.AddOnTheFly(task)
	if ts.Scheduler == nil {
		t.Fatal("expected AddOnTheFly to initialize scheduler")
	}
	if got := len(ts.Scheduler.Jobs()); got != 1 {
		t.Fatalf("expected one scheduled job, got %d", got)
	}

	if err := ts.CancelTask(task.ID); err != nil {
		t.Fatal(err)
	}
	if got := len(ts.Scheduler.Jobs()); got != 0 {
		t.Fatalf("expected no scheduled jobs after cancel, got %d", got)
	}
}

func TestTaskSchedulerStopUnblocksStart(t *testing.T) {
	baseName := fmt.Sprintf("sched_stop_%d", time.Now().UnixNano())
	ds, _ := newSchedulerTestStore(t, baseName)
	ts := &TaskScheduler{
		Volatile:  cache.NewDevCache(logger.Get(config.LoadConfig())),
		DataStore: ds,
		Log:       logger.Get(config.LoadConfig()),
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		ts.Start()
	}()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if ts.Scheduler != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if ts.Scheduler == nil {
		t.Fatal("expected scheduler to start")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := ts.Stop(ctx); err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for scheduler shutdown")
	}
}

func TestTaskSchedulerStopBeforeStart(t *testing.T) {
	ts := &TaskScheduler{Log: logger.Get(config.LoadConfig())}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := ts.Stop(ctx); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		ts.Start()
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for stopped scheduler start to return")
	}
}

func TestTaskSchedulerDoesNotRunCronTaskOnStart(t *testing.T) {
	baseName := fmt.Sprintf("sched_no_start_%d", time.Now().UnixNano())
	ds, _ := newSchedulerTestStore(t, baseName)

	fn := model.ExecData{
		FunctionName: "not-on-start",
		TriggerTopic: "schedule",
		Code: `function handle(channel, type, data) {
			var result = create("unexpected_start_runs", { ok: true });
			if (!result.ok) {
				throw new Error(result.content);
			}
		}`,
		Version: 1,
	}
	if _, err := ds.AddFunction(baseName, fn); err != nil {
		t.Fatal(err)
	}

	task := model.Task{
		Name:     "not-on-start-task",
		Type:     model.TaskTypeFunction,
		Value:    fn.FunctionName,
		Interval: "0 2 * * *",
		BaseName: baseName,
	}
	taskID, err := ds.AddTask(baseName, task)
	if err != nil {
		t.Fatal(err)
	}
	task.ID = taskID

	ts := &TaskScheduler{
		Volatile:  cache.NewDevCache(logger.Get(config.LoadConfig())),
		DataStore: ds,
		Log:       logger.Get(config.LoadConfig()),
	}
	ts.AddOnTheFly(task)
	ts.Scheduler.Start()
	defer func() {
		if err := ts.Scheduler.Shutdown(); err != nil {
			t.Fatal(err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	storedTask, err := ds.GetTask(baseName, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !storedTask.LastRun.IsZero() {
		t.Fatalf("expected cron task not to run on scheduler start, got last run %s", storedTask.LastRun)
	}
}

func newSchedulerTestStore(t *testing.T, baseName string) (database.Persister, model.Auth) {
	t.Helper()

	ds := memory.New(func(model.Auth, string, string, string, interface{}) {})
	tenant, err := ds.CreateTenant(model.Tenant{ID: baseName, Email: baseName + "@test.com"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ds.CreateDatabase(model.DatabaseConfig{
		ID:       baseName,
		TenantID: tenant.ID,
		Name:     baseName,
		IsActive: true,
		Created:  time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	accountID, err := ds.CreateAccount(baseName, baseName+"@test.com")
	if err != nil {
		t.Fatal(err)
	}
	user := model.User{
		AccountID: accountID,
		Email:     baseName + "@test.com",
		Token:     "root-token-" + baseName,
		Role:      100,
	}
	userID, err := ds.CreateUser(baseName, user)
	if err != nil {
		t.Fatal(err)
	}

	return ds, model.Auth{
		AccountID: accountID,
		UserID:    userID,
		Email:     user.Email,
		Role:      user.Role,
		Token:     user.Token,
	}
}

func waitForFunctionHistory(t *testing.T, ds database.Persister, baseName, functionName string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		fn, err := ds.GetFunctionByName(baseName, functionName)
		if err != nil {
			t.Fatal(err)
		}
		if len(fn.History) > 0 {
			if fn.LastRun.IsZero() {
				t.Fatal("expected scheduled function to set last run time")
			}
			if !fn.History[len(fn.History)-1].Success {
				t.Fatalf("expected successful function execution history, got %+v", fn.History[len(fn.History)-1])
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for scheduled function execution history")
}
