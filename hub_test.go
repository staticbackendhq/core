package staticbackend

import (
	"testing"
	"time"

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

	msg := Command{
		SID:  id,
		Type: MsgTypeAuth,
		Data: adminToken,
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

	msg := Command{
		SID:  id1,
		Type: MsgTypeAuth,
		Data: adminToken,
	}

	reply1 := sendReceiveWS(t, sck1, msg)
	if reply1.Type != MsgTypeToken {
		t.Fatalf("expected auth token, got %v", reply1)
	}

	token1 := reply1.Data

	msg = Command{
		SID:   id1,
		Type:  MsgTypeJoin,
		Data:  channel,
		Token: token1,
	}

	reply1 = sendReceiveWS(t, sck1, msg)
	if reply1.Type != MsgTypeJoined {
		t.Fatalf("expected to join the channel, got %v", reply1)
	}

	msg = Command{
		SID:  id2,
		Type: MsgTypeAuth,
		Data: userToken,
	}

	reply2 := sendReceiveWS(t, sck2, msg)
	if reply2.Type != MsgTypeToken {
		t.Fatalf("expected auth to return a token, got %v", reply2)
	}

	token2 := reply2.Data

	msg = Command{
		SID:   id2,
		Type:  MsgTypeJoin,
		Data:  channel,
		Token: token2,
	}

	reply2 = sendReceiveWS(t, sck2, msg)
	if reply2.Type != MsgTypeJoined {
		t.Fatalf("expected to join the channel, got %v", reply2)
	}

	time.Sleep(300 * time.Millisecond)

	// sending a msg to channel from sck1 should be sent to both socket
	msg.SID = id1
	msg.Type = MsgTypeChanIn
	msg.Data = "hello sck1 and sck2"
	msg.Channel = channel
	msg.Token = token1

	reply1 = sendReceiveWS(t, sck1, msg)
	if reply1.Type != MsgTypeOk {
		t.Fatalf(`expected type to be %s" got %s`, MsgTypeOk, reply1.Type)
	}

	time.Sleep(300 * time.Millisecond)

	// we manually read sck2, no need to send to receive
	if err := sck2.ReadJSON(&reply2); err != nil {
		t.Fatal(err)
	} else if reply2.Type != MsgTypeChanOut || reply2.Data != msg.Data {
		t.Fatalf(`expected type to be "%s" got %s and data %s`, MsgTypeChanOut, reply2.Type, reply2.Data)
	}
}

func TestWebSocketDBEvents(t *testing.T) {
	channel := "db-test"

	sck1, id1 := newWsConn(t)
	defer sck1.Close()

	msg := Command{
		SID:  id1,
		Type: MsgTypeAuth,
		Data: adminToken,
	}

	reply1 := sendReceiveWS(t, sck1, msg)
	if reply1.Type != MsgTypeToken {
		t.Fatalf("auth reply type expected %s got %s", MsgTypeToken, reply1.Type)
	}

	token1 := reply1.Data

	msg.Type = MsgTypeJoin
	msg.Data = channel
	msg.Token = token1

	reply1 = sendReceiveWS(t, sck1, msg)
	if reply1.Type != MsgTypeJoined {
		t.Fatalf("expected to join the channel, got %v", reply1)
	}

	sck2, id2 := newWsConn(t)
	defer sck2.Close()

	msg = Command{
		SID:  id2,
		Type: MsgTypeAuth,
		Data: userToken,
	}

	reply2 := sendReceiveWS(t, sck2, msg)
	if reply2.Type != MsgTypeToken {
		t.Fatalf("auth reply type expected %s got %s", MsgTypeToken, reply2.Type)
	}

	token2 := reply2.Data

	msg.Type = MsgTypeJoin
	msg.Data = channel
	msg.Token = token2

	reply2 = sendReceiveWS(t, sck2, msg)
	if reply2.Type != MsgTypeJoined {
		t.Fatalf("expected to join the channel, got %v", reply2)
	}

	time.Sleep(350 * time.Millisecond)

	// we create a doc which should trigger a message to the db-test channel
	task := Task{
		Title:   "websocket test",
		Created: time.Now(),
	}
	resp := dbPost(t, database.add, "test", task)
	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	time.Sleep(300 * time.Millisecond)

	// manual read
	var eventMsg Command
	if err := sck1.ReadJSON(&eventMsg); err != nil {
		t.Error(err)
	} else if eventMsg.Type != MsgTypeDBCreated {
		t.Errorf("expected msg type to be %s to %s", MsgTypeDBCreated, eventMsg.Type)
	}

	if err := sck2.ReadJSON(&eventMsg); err != nil {
		t.Error(err)
	} else if eventMsg.Type != MsgTypeDBCreated {
		t.Errorf("expected msg type to be %s to %s", MsgTypeDBCreated, eventMsg.Type)
	}
}

func TestWebSocketDBPermission(t *testing.T) {
	channel := "db-permtest_700_"

	sck1, id1 := newWsConn(t)
	defer sck1.Close()

	msg := Command{
		SID:  id1,
		Type: MsgTypeAuth,
		Data: adminToken,
	}

	reply1 := sendReceiveWS(t, sck1, msg)
	if reply1.Type != MsgTypeToken {
		t.Fatalf("auth reply type expected %s got %s", MsgTypeToken, reply1.Type)
	}

	token1 := reply1.Data

	msg.Type = MsgTypeJoin
	msg.Data = channel
	msg.Token = token1

	reply1 = sendReceiveWS(t, sck1, msg)
	if reply1.Type != MsgTypeJoined {
		t.Fatalf("expected to join the channel, got %v", reply1)
	}

	sck2, id2 := newWsConn(t)
	defer sck2.Close()

	msg = Command{
		SID:  id2,
		Type: MsgTypeAuth,
		Data: userToken,
	}

	reply2 := sendReceiveWS(t, sck2, msg)
	if reply2.Type != MsgTypeToken {
		t.Fatalf("auth reply type expected %s got %s", MsgTypeToken, reply2.Type)
	}

	token2 := reply2.Data

	msg.Type = MsgTypeJoin
	msg.Data = channel
	msg.Token = token2

	reply2 = sendReceiveWS(t, sck2, msg)
	if reply2.Type != MsgTypeJoined {
		t.Fatalf("expected to join the channel, got %v", reply2)
	}

	time.Sleep(350 * time.Millisecond)

	// we create a doc which should trigger a message to the db-test channel
	task := Task{
		Title:   "websocket test",
		Created: time.Now(),
	}
	resp := dbPost(t, database.add, "permtest_700_", task)
	if resp.StatusCode > 299 {
		t.Fatal(GetResponseBody(t, resp))
	}

	time.Sleep(300 * time.Millisecond)

	// manual read
	var eventMsg Command
	if err := sck1.ReadJSON(&eventMsg); err != nil {
		t.Error(err)
	} else if eventMsg.Type != MsgTypeDBCreated {
		t.Errorf("expected msg type to be %s to %s", MsgTypeDBCreated, eventMsg.Type)
	}

	go func() {
		if err := sck2.ReadJSON(&eventMsg); err != nil {
			t.Log("normal to get an error since we've manually close", err)
		} else if eventMsg.Type == MsgTypeDBCreated {
			t.Error("sck2 should not receive the created message")
		}
	}()

	// the second socket should not receive anything
	time.AfterFunc(600*time.Millisecond, func() {
		sck2.Close()
	})

	time.Sleep(650 * time.Millisecond)
}
