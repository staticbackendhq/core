package main

import (
	"testing"

	"github.com/gorilla/websocket"
)

func newWsConn(t *testing.T) (*websocket.Conn, string) {
	sck, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal("cannot connect WebSocket", err)
	}

	var initMsg Command
	if err := sck.ReadJSON(&initMsg); err != nil {
		t.Fatal(err)
	}

	return sck, initMsg.Data
}

func sendReceiveWS(t *testing.T, sck *websocket.Conn, msg Command) Command {
	if err := sck.WriteJSON(msg); err != nil {
		t.Fatal("error writing JSON to WebSocket", err)
	}

	if err := sck.ReadJSON(&msg); err != nil {
		t.Fatal("error reading JSON from WebSocket", err)
	}

	return msg
}

func TestWebSocketConnection(t *testing.T) {
	sck, id := newWsConn(t)
	defer sck.Close()

	msg := Command{
		SID:  id,
		Type: MsgTypeEcho,
		Data: "test",
	}

	msg = sendReceiveWS(t, sck, msg)
	if msg.Data != "echo: test" {
		t.Fatalf(`expected msg to be "echo: test" got %s`, msg.Data)
	}
}

func TestWebSocketAuth(t *testing.T) {
	sck, id := newWsConn(t)
	defer sck.Close()

	// fake if they authenticated
	tokens["unit-test-auth-key"] = Auth{}

	msg := Command{
		SID:  id,
		Type: MsgTypeAuth,
		Data: "unit-test-auth-key",
	}
	msg = sendReceiveWS(t, sck, msg)
	if msg.Type != MsgTypeToken {
		t.Errorf(`expected "%s" as reply got %s`, MsgTypeToken, msg.Type)
	}
}

func TestWebSocketChannel(t *testing.T) {
	channel := "unittest"

	sck1, id1 := newWsConn(t)
	defer sck1.Close()

	sck2, id2 := newWsConn(t)
	defer sck2.Close()

	// fake that they are signed in
	tokens["sck1"] = Auth{}
	tokens["sck2"] = Auth{}

	msg := Command{
		SID:  id1,
		Type: MsgTypeJoin,
		Data: channel,
	}

	reply1 := sendReceiveWS(t, sck1, msg)
	if reply1.Type != MsgTypeJoined {
		t.Fatalf("expected to join the channel, got %v", reply1)
	}

	msg.SID = id2
	reply2 := sendReceiveWS(t, sck2, msg)
	if reply1.Type != MsgTypeJoined {
		t.Fatalf("expected to join the channel, got %v", reply1)
	}

	// sending a msg to channel from sck1 should be sent to both socket
	msg.SID = id1
	msg.Type = MsgTypeChanIn
	msg.Data = "hello sck1 and sck2"
	msg.Channel = channel

	reply1 = sendReceiveWS(t, sck1, msg)
	if reply1.Type != MsgTypeOk {
		t.Fatalf(`expected type to be %s" got %s`, MsgTypeOk, reply1.Type)
	}

	// we manually read sck2, no need to send to receive
	if err := sck2.ReadJSON(&reply2); err != nil {
		t.Fatal(err)
	} else if reply2.Type != MsgTypeChanOut || reply2.Data != msg.Data {
		t.Fatalf(`expected type to be "%s" got %s and data %s`, MsgTypeChanOut, reply2.Type, reply2.Data)
	}
}
