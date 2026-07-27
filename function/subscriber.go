package function

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/model"
)

type Subscriber struct {
	PubSub            cache.Volatilizer
	GetExecEnv        func(msg model.Command) (*ExecutionEnvironment, error)
	IsPrimaryInstance bool

	relax sync.Map
}

// Start starts the system event subscription.
// This channel is responsible of executing functions that match the
// topic/trigger
func (sub *Subscriber) Start() {
	sub.StartContext(context.Background())
}

// StartContext starts the system event subscription until ctx is canceled.
func (sub *Subscriber) StartContext(ctx context.Context) {
	receiver := make(chan model.Command)
	closeSub := make(chan bool)
	var wg sync.WaitGroup

	go sub.PubSub.Subscribe(receiver, "", "sbsys", closeSub)

	for {
		select {
		case msg := <-receiver:
			// only handle function execution on the primary instance
			// otherwise it would cause duplication work.
			if sub.IsPrimaryInstance {
				wg.Add(1)
				go func() {
					defer wg.Done()
					sub.process(msg)
				}()
			}
		case <-closeSub:
			slog.Info("system event channel closed")
		case <-ctx.Done():
			close(closeSub)
			wg.Wait()
			return
		}
	}
}

func (sub *Subscriber) process(msg model.Command) {
	var wg sync.WaitGroup
	defer wg.Wait()

	switch msg.Type {
	case model.MsgTypeChanOut,
		model.MsgTypeDBCreated,
		model.MsgTypeDBUpdated,
		model.MsgTypeDBDeleted,
		model.MsgTypeTelemetryLongRequest:
		sub.handleRealtimeEvents(msg, &wg)
	default:
		// for user triggered events, we enforce a max of 5 msg / 60 secs
		v, ok := sub.relax.Load(msg.Auth.UserID)
		if !ok {
			v = int64(0)

			go func(key string) {
				time.Sleep(60 * time.Second)
				sub.relax.Delete(key)
			}(msg.Auth.UserID)
		}

		n, ok := v.(int64)
		if !ok {
			slog.Warn("subscriber.process(): unable to cast value into int64", "value", v)
			return
		} else if n >= 5 {
			slog.Warn("user exceeded amount of allowed message in 60 seconds", "count", n)
			//TODO: This silently returns, should this app owner get some
			// notification about their users flooding the system?
			return
		}

		n += 1
		sub.relax.Store(msg.Auth.UserID, n)

		sub.handleRealtimeEvents(msg, &wg)
	}
}

func (sub *Subscriber) handleRealtimeEvents(msg model.Command, wg *sync.WaitGroup) {
	exe, err := sub.GetExecEnv(msg)
	if err != nil {
		slog.Error("cannot retrieve base from token", "token", msg.Token, "error", err)
		return
	}

	var ids []string

	// for msg type error, we do nothing
	if msg.Type == model.MsgTypeError {
		slog.Error("receiving msg of type error")
		return
	}

	key := fmt.Sprintf("%s:%s", exe.BaseName, msg.Channel)
	if err := sub.PubSub.GetTyped(key, &ids); err != nil {
		funcs, err := exe.DataStore.ListFunctionsByTrigger(exe.BaseName, msg.Channel)
		if err != nil {
			slog.Error("error getting functions by trigger", "error", err)
			slog.Debug("channel and message type", "channel", msg.Channel, "type", msg.Type)
			return
		}

		for _, fn := range funcs {
			if err := sub.PubSub.SetTyped("fn_"+fn.ID, fn); err != nil {
				slog.Error("error adding function to cache", "error", err)
				return
			}

			ids = append(ids, fn.ID)
		}

		if err := sub.PubSub.SetTyped(key, ids); err != nil {
			slog.Error("unable to publish message", "error", err)
		}
	}

	for _, id := range ids {
		var fn model.ExecData
		if err := sub.PubSub.GetTyped("fn_"+id, &fn); err != nil {
			slog.Error("error getting function out of cache", "error", err)
			return
		}
		fn, err = exe.DataStore.GetFunctionForExecution(exe.BaseName, fn.FunctionName)
		if err != nil {
			slog.Error("error getting function for execution", "error", err)
			return
		}

		exe.Data = fn
		wg.Add(1)
		go func(ex *ExecutionEnvironment) {
			defer wg.Done()
			if err := ex.Execute(msg); err != nil {
				slog.Error("executing function failed", "function", ex.Data.FunctionName, "error", err)
			}
		}(exe)
	}
}
