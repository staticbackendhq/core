package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/staticbackendhq/core/internal"

	"github.com/google/uuid"
)

type Validator func(context.Context, string) (string, error)

type ConnectionData struct {
	ctx      context.Context
	messages chan internal.Command
}

type Broker struct {
	Broadcast          chan internal.Command
	newConnections     chan ConnectionData
	closingConnections chan chan internal.Command
	clients            map[chan internal.Command]string
	ids                map[string]chan internal.Command
	conf               map[string]context.Context
	subscriptions      map[string][]chan bool
	validateAuth       Validator

	pubsub internal.PubSuber
}

func NewBroker(v Validator, pubsub internal.PubSuber) *Broker {
	b := &Broker{
		Broadcast:          make(chan internal.Command, 1),
		newConnections:     make(chan ConnectionData),
		closingConnections: make(chan chan internal.Command),
		clients:            make(map[chan internal.Command]string),
		ids:                make(map[string]chan internal.Command),
		conf:               make(map[string]context.Context),
		subscriptions:      make(map[string][]chan bool),
		validateAuth:       v,
		pubsub:             pubsub,
	}

	go b.start()

	return b
}

func (b *Broker) start() {
	for {
		select {
		case data := <-b.newConnections:
			id, err := uuid.NewUUID()
			if err != nil {
				log.Println(err)
			}

			b.clients[data.messages] = id.String()
			b.ids[id.String()] = data.messages
			b.conf[id.String()] = data.ctx

			msg := internal.Command{
				Type: internal.MsgTypeInit,
				Data: id.String(),
			}

			data.messages <- msg
		case c := <-b.closingConnections:
			b.unsub(c)
		case msg := <-b.Broadcast:
			clients, payload := b.getTargets(msg)
			for _, c := range clients {
				c <- payload
			}
		}
	}
}

func (b *Broker) unsub(c chan internal.Command) {
	defer delete(b.clients, c)

	id, ok := b.clients[c]
	if !ok {
		fmt.Println("cannot find connection id")
	}

	subs, ok := b.subscriptions[id]
	if ok {
		for _, ch := range subs {
			ch <- true
		}
	}

	delete(b.ids, id)
}

func (b *Broker) Accept(w http.ResponseWriter, r *http.Request) {
	// check if writer handles flushing
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is unsupported with your connection.", http.StatusBadRequest)
		return
	}

	// set headers related to event streaming
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	//w.Header().Set("Access-Control-Allow-Origin", "*")

	// each connection has their own message channel
	messages := make(chan internal.Command)
	data := ConnectionData{
		ctx:      r.Context(),
		messages: messages,
	}
	b.newConnections <- data

	// make sure we'r removing this connection
	// when the handler completes.
	defer func() {
		b.closingConnections <- messages
	}()

	// handles the client-side disconnection
	notify := w.(http.CloseNotifier).CloseNotify()
	go func() {
		<-notify
		b.closingConnections <- messages
	}()

	// broadcast messages
	for {
		// write Server Sent Event data
		msg := <-messages
		b, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("error converting to JSON", err)
			continue
		}
		fmt.Fprintf(w, "data: %s\n\n", b)

		// flush immediately.
		flusher.Flush()
	}
}

func (b *Broker) getTargets(msg internal.Command) (sockets []chan internal.Command, payload internal.Command) {
	var sender chan internal.Command

	if msg.SID != internal.SystemID {
		s, ok := b.ids[msg.SID]
		if !ok {
			fmt.Println("cannot find sender socket", msg.SID)
			return
		}
		sender = s
		sockets = append(sockets, sender)
	}

	switch msg.Type {
	case internal.MsgTypeEcho:
		payload = msg
		payload.Data = "echo: " + msg.Data
	case internal.MsgTypeAuth:
		ctx, ok := b.conf[msg.SID]
		if !ok {
			payload = internal.Command{Type: internal.MsgTypeError, Data: "invalid request"}
			return
		}

		if _, err := b.validateAuth(ctx, msg.Data); err != nil {
			payload = internal.Command{Type: internal.MsgTypeError, Data: "invalid token"}
			return
		}

		payload = internal.Command{Type: internal.MsgTypeToken, Data: msg.Data}
	case internal.MsgTypeJoin:
		subs, ok := b.subscriptions[msg.SID]
		if !ok {
			subs = make([]chan bool, 0)
		}

		closesub := make(chan bool)

		subs = append(subs, closesub)
		b.subscriptions[msg.SID] = subs

		go b.pubsub.Subscribe(sender, msg.Token, msg.Data, closesub)

		joinedMsg := internal.Command{
			Type:    internal.MsgTypeJoined,
			Data:    msg.SID,
			Channel: msg.Data,
		}
		// make sure the subscription had time to kick-off
		go func(m internal.Command) {
			time.Sleep(250 * time.Millisecond)
			b.pubsub.Publish(joinedMsg)
		}(joinedMsg)

		payload = internal.Command{Type: internal.MsgTypeOk, Data: msg.Data}
	case internal.MsgTypePresence:
		v, err := b.pubsub.Get(msg.Data)
		if err != nil {
			//TODO: Make sure it's because the channel key does not exists
			v = "0"
		}

		payload = internal.Command{Type: internal.MsgTypePresence, Data: v}
	case internal.MsgTypeChanIn:
		if len(msg.Channel) == 0 {
			payload = internal.Command{Type: internal.MsgTypeError, Data: "no channel was specified"}
			return
		} else if strings.HasPrefix(strings.ToLower(msg.Channel), "db-") {
			payload = internal.Command{
				Type: internal.MsgTypeError,
				Data: "you cannot write to database channel",
			}
			return
		}

		go b.pubsub.Publish(msg)
		//go b.Publish(msg, msg.Channel)

		payload = internal.Command{Type: internal.MsgTypeOk}
	default:
		payload.Type = internal.MsgTypeError
		payload.Data = fmt.Sprintf(`%s command not found`, msg.Type)
	}

	return
}
