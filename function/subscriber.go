package function

import (
	"fmt"
	"log"

	"github.com/staticbackendhq/core/internal"
)

type Subscriber struct {
	PubSub     internal.PubSuber
	GetExecEnv func(token string) (ExecutionEnvironment, error)
}

// Start starts the system event subscription.
// This channel is responsible of executing functions that match the
// topic/trigger
func (sub *Subscriber) Start() {
	receiver := make(chan internal.Command)
	close := make(chan bool)

	go sub.PubSub.Subscribe(receiver, "", "sbsys", close)

	for {
		select {
		case msg := <-receiver:
			go sub.process(msg)
		case <-close:
			log.Println("system event channel closed?!?")
		}
	}
}

func (sub *Subscriber) process(msg internal.Command) {
	switch msg.Type {
	case internal.MsgTypeChanOut,
		internal.MsgTypeDBCreated,
		internal.MsgTypeDBUpdated,
		internal.MsgTypeDBDeleted:
		sub.handleRealtimeEvents(msg)
	}
}

func (sub *Subscriber) handleRealtimeEvents(msg internal.Command) {
	exe, err := sub.GetExecEnv(msg.Token)
	if err != nil {
		log.Println("cannot retrieve base from token", msg.Token)
		log.Println(err)
		return
	}

	var ids []string

	key := fmt.Sprintf("%s:%s", exe.BaseName, msg.Type)
	if err := sub.PubSub.GetTyped(key, &ids); err != nil {
		funcs, err := exe.DataStore.ListFunctionsByTrigger(exe.BaseName, msg.Type)
		if err != nil {
			log.Println("error getting functions by trigger: ", err)
			return
		}

		for _, fn := range funcs {
			if err := sub.PubSub.SetTyped("fn_"+fn.ID, fn); err != nil {
				log.Println("error adding function  to cache: ", err)
				return
			}

			ids = append(ids, fn.ID)
		}

		sub.PubSub.SetTyped(key, ids)
	}

	for _, id := range ids {
		var fn internal.ExecData
		if err := sub.PubSub.GetTyped("fn_"+id, &fn); err != nil {
			log.Println("error getting function out of cache: ", err)
			return
		}

		exe.Data = fn
		go func(ex ExecutionEnvironment) {
			if err := ex.Execute(msg); err != nil {
				log.Printf(`executing "%s" function failed: %v"`, ex.Data.FunctionName, err)
			}
		}(exe)
	}
}
