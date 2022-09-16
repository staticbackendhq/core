package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/model"

	"github.com/google/uuid"
)

type Validator func(context.Context, string) (string, error)

type ConnectionData struct {
	ctx      context.Context
	messages chan model.Command
}

type Broker struct {
	Broadcast          chan model.Command
	newConnections     chan ConnectionData
	closingConnections chan chan model.Command
	clients            map[chan model.Command]string
	ids                map[string]chan model.Command
	conf               map[string]context.Context
	subscriptions      map[string][]chan bool
	validateAuth       Validator

	pubsub cache.Volatilizer

	log *logger.Logger
}

func NewBroker(v Validator, pubsub cache.Volatilizer, log *logger.Logger) *Broker {
	b := &Broker{
		Broadcast:          make(chan model.Command, 1),
		newConnections:     make(chan ConnectionData),
		closingConnections: make(chan chan model.Command),
		clients:            make(map[chan model.Command]string),
		ids:                make(map[string]chan model.Command),
		conf:               make(map[string]context.Context),
		subscriptions:      make(map[string][]chan bool),
		validateAuth:       v,
		pubsub:             pubsub,
		log:                log,
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
				b.log.Error().Err(err)
			}

			b.clients[data.messages] = id.String()
			b.ids[id.String()] = data.messages
			b.conf[id.String()] = data.ctx

			msg := model.Command{
				Type: model.MsgTypeInit,
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

func (b *Broker) unsub(c chan model.Command) {
	defer delete(b.clients, c)

	id, ok := b.clients[c]
	if !ok {
		b.log.Info().Msg("cannot find connection id")
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
	messages := make(chan model.Command)
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
		bytes, err := json.Marshal(msg)
		if err != nil {
			b.log.Warn().Err(err).Msg("error converting to JSON")

			continue
		}

		fmt.Fprintf(w, "data: %s\n\n", bytes)

		// flush immediately.
		flusher.Flush()
	}
}

func (b *Broker) getTargets(msg model.Command) (sockets []chan model.Command, payload model.Command) {
	var sender chan model.Command

	if msg.SID != model.SystemID {
		s, ok := b.ids[msg.SID]
		if !ok {
			b.log.Info().Msgf("cannot find sender socket: %d", msg.SID)
			return
		}
		sender = s
		sockets = append(sockets, sender)
	}

	switch msg.Type {
	case model.MsgTypeEcho:
		payload = msg
		payload.Data = "echo: " + msg.Data
	case model.MsgTypeAuth:
		ctx, ok := b.conf[msg.SID]
		if !ok {
			payload = model.Command{Type: model.MsgTypeError, Data: "invalid request"}
			return
		}

		if _, err := b.validateAuth(ctx, msg.Data); err != nil {
			payload = model.Command{Type: model.MsgTypeError, Data: "invalid token"}
			return
		}

		payload = model.Command{Type: model.MsgTypeToken, Data: msg.Data}
	case model.MsgTypeJoin:
		subs, ok := b.subscriptions[msg.SID]
		if !ok {
			subs = make([]chan bool, 0)
		}

		closesub := make(chan bool)

		subs = append(subs, closesub)
		b.subscriptions[msg.SID] = subs

		go b.pubsub.Subscribe(sender, msg.Token, msg.Data, closesub)

		joinedMsg := model.Command{
			Type:    model.MsgTypeJoined,
			Data:    msg.SID,
			Channel: msg.Data,
		}
		// make sure the subscription had time to kick-off
		go func(m model.Command) {
			time.Sleep(250 * time.Millisecond)
			b.pubsub.Publish(joinedMsg)
		}(joinedMsg)

		payload = model.Command{Type: model.MsgTypeOk, Data: msg.Data}
	case model.MsgTypePresence:
		v, err := b.pubsub.Get(msg.Data)
		if err != nil {
			//TODO: Make sure it's because the channel key does not exists
			v = "0"
		}

		payload = model.Command{Type: model.MsgTypePresence, Data: v}
	case model.MsgTypeChanIn:
		if len(msg.Channel) == 0 {
			payload = model.Command{Type: model.MsgTypeError, Data: "no channel was specified"}
			return
		} else if strings.HasPrefix(strings.ToLower(msg.Channel), "db-") {
			payload = model.Command{
				Type: model.MsgTypeError,
				Data: "you cannot write to database channel",
			}
			return
		}

		go b.pubsub.Publish(msg)
		//go b.Publish(msg, msg.Channel)

		payload = model.Command{Type: model.MsgTypeOk}
	default:
		payload.Type = model.MsgTypeError
		payload.Data = fmt.Sprintf(`%s command not found`, msg.Type)
	}

	return
}
