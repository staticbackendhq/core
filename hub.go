package staticbackend

import (
	"fmt"
	"strings"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/internal"

	"github.com/gbrlsnchs/jwt/v3"
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
	broadcast chan internal.Command

	// Register requests from the clients.
	register chan *Socket

	// Unregister requests from clients.
	unregister chan *Socket

	// Cache used for keys and pub/sub (Redis)
	volatile *cache.Cache
}

func newHub(c *cache.Cache) *Hub {
	return &Hub{
		broadcast:  make(chan internal.Command),
		register:   make(chan *Socket),
		unregister: make(chan *Socket),
		sockets:    make(map[*Socket]string),
		ids:        make(map[string]*Socket),
		channels:   make(map[*Socket][]chan bool),
		volatile:   c,
	}
}

func (h *Hub) run() {
	for {
		select {
		case sck := <-h.register:
			h.sockets[sck] = sck.id
			h.ids[sck.id] = sck

			cmd := internal.Command{
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
				//time.AfterFunc(500*time.Millisecond, func() {
				close(sck.send)
				//})
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

func (h *Hub) getTargets(msg internal.Command) (sockets []*Socket, payload internal.Command) {
	sender, ok := h.ids[msg.SID]
	if !ok {
		return
	}

	switch msg.Type {
	case internal.MsgTypeEcho:
		sockets = append(sockets, sender)
		payload = msg
		payload.Data = "echo: " + msg.Data
	case internal.MsgTypeAuth:
		sockets = append(sockets, sender)
		var pl internal.JWTPayload
		if _, err := jwt.Verify([]byte(msg.Data), internal.HashSecret, &pl); err != nil {
			payload = internal.Command{Type: internal.MsgTypeError, Data: "invalid token"}
			return
		}

		var a internal.Auth
		if err := volatile.GetTyped(pl.Token, &a); err != nil {
			payload = internal.Command{Type: internal.MsgTypeError, Data: "invalid token"}
		} else {
			payload = internal.Command{Type: internal.MsgTypeToken, Data: pl.Token}
		}
	case internal.MsgTypeJoin:
		subs, ok := h.channels[sender]
		if !ok {
			subs = make([]chan bool, 0)
		}

		closeSubChan := make(chan bool)
		subs = append(subs, closeSubChan)

		go h.volatile.Subscribe(sender.send, msg.Token, msg.Data, closeSubChan)

		sockets = append(sockets, sender)
		payload = internal.Command{Type: internal.MsgTypeJoined, Data: msg.Data}
	case internal.MsgTypeChanIn:
		sockets = append(sockets, sender)

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

		if err := h.volatile.Publish(msg); err != nil {
			payload = internal.Command{Type: internal.MsgTypeError, Data: "unable to send your message"}
			return
		}

		payload = internal.Command{Type: internal.MsgTypeOk}
	default:
		sockets = append(sockets, sender)

		payload.Type = internal.MsgTypeError
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
