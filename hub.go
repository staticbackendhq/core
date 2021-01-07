package main

import (
	"fmt"

	"github.com/gorilla/websocket"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	sockets map[*Socket]string

	// Reverse ID => socket
	ids map[string]*Socket

	// Socket's subscribed channels
	channels map[*Socket][]chan bool

	// Inbound messages from the clients.
	broadcast chan Command

	// Register requests from the clients.
	register chan *Socket

	// Unregister requests from clients.
	unregister chan *Socket

	// Cache used for keys and pub/sub (Redis)
	cache *Cache
}

type Command struct {
	SID     string `json:"sid"`
	Type    string `json:"type"`
	Data    string `json:"data"`
	Channel string `json:"channel"`
}

func newHub(c *Cache) *Hub {
	return &Hub{
		broadcast:  make(chan Command),
		register:   make(chan *Socket),
		unregister: make(chan *Socket),
		sockets:    make(map[*Socket]string),
		ids:        make(map[string]*Socket),
		channels:   make(map[*Socket][]chan bool),
		cache:      c,
	}
}

func (h *Hub) run() {
	for {
		select {
		case sck := <-h.register:
			h.sockets[sck] = sck.id
			h.ids[sck.id] = sck

			cmd := Command{
				Type: "init",
				Data: sck.id,
			}
			sck.send <- cmd

		case sck := <-h.unregister:
			if _, ok := h.sockets[sck]; ok {
				h.unsub(sck)
				delete(h.sockets, sck)
				delete(h.ids, sck.id)
				delete(h.channels, sck)
				close(sck.send)
			}
		case msg := <-h.broadcast:
			sockets, p := h.getTargets(msg)
			for _, sck := range sockets {
				select {
				case sck.send <- p:
				default:
					h.unsub(sck)
					close(sck.send)
					delete(h.ids, msg.SID)
					delete(h.sockets, sck)
					delete(h.channels, sck)
				}
			}
		}
	}
}

const (
	MsgTypeError   = "error"
	MsgTypeOk      = "ok"
	MsgTypeEcho    = "echo"
	MsgTypeAuth    = "auth"
	MsgTypeToken   = "token"
	MsgTypeJoin    = "join"
	MsgTypeJoined  = "joined"
	MsgTypeChanIn  = "chan_in"
	MsgTypeChanOut = "chan_out"
)

func (h *Hub) getTargets(msg Command) (sockets []*Socket, payload Command) {
	sender, ok := h.ids[msg.SID]
	if !ok {
		return
	}

	switch msg.Type {
	case MsgTypeEcho:
		sockets = append(sockets, sender)
		payload = msg
		payload.Data = "echo: " + msg.Data
	case MsgTypeAuth:
		sockets = append(sockets, sender)
		_, ok := tokens[msg.Data]
		if !ok {
			payload = Command{Type: MsgTypeError, Data: "invalid token"}
		} else {
			payload = Command{Type: MsgTypeToken, Data: msg.Data}
		}
	case MsgTypeJoin:
		subs, ok := h.channels[sender]
		if !ok {
			subs = make([]chan bool, 0)
		}

		closeSubChan := make(chan bool)
		subs = append(subs, closeSubChan)

		go h.cache.Subscribe(sender.send, msg.Data, closeSubChan)

		sockets = append(sockets, sender)
		payload = Command{Type: MsgTypeJoined, Data: msg.Data}
	case MsgTypeChanIn:
		sockets = append(sockets, sender)

		if len(msg.Channel) == 0 {
			payload = Command{Type: MsgTypeError, Data: "no channel was specified"}
			return
		}

		if err := h.cache.Publish(msg); err != nil {
			payload = Command{Type: MsgTypeError, Data: "unable to send your message"}
			return
		}

		payload = Command{Type: MsgTypeOk}
	default:
		sockets = append(sockets, sender)

		payload.Type = MsgTypeError
		payload.Data = fmt.Sprintf(`%s command not found`, msg.Type)
	}

	return
}

func (h *Hub) join(scksck *websocket.Conn, channel string) {

}

func (h *Hub) unsub(sck *Socket) {
	subs, ok := h.channels[sck]
	if !ok {
		return
	}

	for _, sub := range subs {
		sub <- true
		close(sub)
	}
}
