package function

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/database/memory"
	"github.com/staticbackendhq/core/model"
)

func TestSubscriberDBTriggerRecordsHistoryAndLastRun(t *testing.T) {
	baseName := "subscriber_db_trigger"

	vol := cache.NewDevCache()
	ds := memory.New(vol.PublishDocument)

	fnID, err := ds.AddFunction(baseName, model.ExecData{
		FunctionName: "index-contact",
		TriggerTopic: "db-contacts",
		Code: `function handle(channel, type, data) {
			if (type != "db_created") return;
			log("indexed " + data.name);
		}`,
		Version: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	sub := &Subscriber{
		PubSub: vol,
		GetExecEnv: func(msg model.Command) (*ExecutionEnvironment, error) {
			return &ExecutionEnvironment{
				Auth:      msg.Auth,
				BaseName:  msg.Base,
				DataStore: ds,
				Volatile:  vol,
			}, nil
		},
	}

	var wg sync.WaitGroup
	sub.handleRealtimeEvents(model.Command{
		Channel: "db-contacts",
		Type:    model.MsgTypeDBCreated,
		Data:    `{"id":"contact-1","name":"Ada"}`,
		Base:    baseName,
		Auth: model.Auth{
			AccountID: "account-1",
			UserID:    "user-1",
			Role:      100,
			Token:     "token-1",
		},
	}, &wg)
	wg.Wait()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		fn, err := ds.GetFunctionByID(baseName, fnID)
		if err != nil {
			t.Fatal(err)
		}
		if len(fn.History) == 0 {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		if fn.LastRun.IsZero() {
			t.Fatal("expected db-triggered function to set last run time")
		}
		history := fn.History[len(fn.History)-1]
		if !history.Success {
			t.Fatalf("expected successful function execution history, got %+v", history)
		}
		if !strings.Contains(strings.Join(history.Output, "\n"), "indexed Ada") {
			t.Fatalf("expected function output to include index log, got %+v", history.Output)
		}
		return
	}

	t.Fatal("timed out waiting for db-triggered function execution history")
}

func TestSubscriberTelemetryBypassesUserMessageThrottle(t *testing.T) {
	baseName := "subscriber_telemetry_trigger"

	vol := cache.NewDevCache()
	ds := memory.New(vol.PublishDocument)

	called := false
	sub := &Subscriber{
		PubSub: vol,
		GetExecEnv: func(msg model.Command) (*ExecutionEnvironment, error) {
			called = true
			return &ExecutionEnvironment{
				Auth:      msg.Auth,
				BaseName:  msg.Base,
				DataStore: ds,
				Volatile:  vol,
			}, nil
		},
	}
	sub.relax.Store("user-1", int64(5))

	sub.process(model.Command{
		Channel: model.TelemetryLongRequestChannel,
		Type:    model.MsgTypeTelemetryLongRequest,
		Data:    `{"path":"/db/tasks"}`,
		Base:    baseName,
		Auth: model.Auth{
			AccountID: "account-1",
			UserID:    "user-1",
			Token:     "token-1",
		},
	})

	if !called {
		t.Fatal("expected telemetry event to bypass user message throttle")
	}
}

func TestSubscriberTelemetryTriggerRecordsHistory(t *testing.T) {
	baseName := "subscriber_telemetry_history"

	vol := cache.NewDevCache()
	ds := memory.New(vol.PublishDocument)

	fnID, err := ds.AddFunction(baseName, model.ExecData{
		FunctionName: "capture-slow-request",
		TriggerTopic: model.TelemetryLongRequestChannel,
		Code: `function handle(channel, type, data) {
			if (type != "telemetry_long_request") return;
			log("slow " + data.method + " " + data.path);
		}`,
		Version: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	sub := &Subscriber{
		PubSub: vol,
		GetExecEnv: func(msg model.Command) (*ExecutionEnvironment, error) {
			return &ExecutionEnvironment{
				Auth:      msg.Auth,
				BaseName:  msg.Base,
				DataStore: ds,
				Volatile:  vol,
			}, nil
		},
	}

	var wg sync.WaitGroup
	sub.handleRealtimeEvents(model.Command{
		Channel: model.TelemetryLongRequestChannel,
		Type:    model.MsgTypeTelemetryLongRequest,
		Data:    `{"method":"GET","path":"/db/tasks"}`,
		Base:    baseName,
		Auth: model.Auth{
			AccountID: "account-1",
			UserID:    "user-1",
			Role:      100,
			Token:     "token-1",
		},
	}, &wg)
	wg.Wait()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		fn, err := ds.GetFunctionByID(baseName, fnID)
		if err != nil {
			t.Fatal(err)
		}
		if len(fn.History) == 0 {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		history := fn.History[len(fn.History)-1]
		if !history.Success {
			t.Fatalf("expected successful telemetry function execution history, got %+v", history)
		}
		if !strings.Contains(strings.Join(history.Output, "\n"), "slow GET /db/tasks") {
			t.Fatalf("expected function output to include telemetry path, got %+v", history.Output)
		}
		return
	}

	t.Fatal("timed out waiting for telemetry function execution history")
}

func TestSubscriberStartContextStopsOnCancel(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	sub := &Subscriber{
		PubSub: cache.NewDevCache(),
		GetExecEnv: func(msg model.Command) (*ExecutionEnvironment, error) {
			t.Fatal("unexpected message processing")
			return nil, nil
		},
	}

	go func() {
		defer close(done)
		sub.StartContext(ctx)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for subscriber shutdown")
	}
}
