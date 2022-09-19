package cache

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/model"
)

var (
	redisCache *Cache
	devCache   *CacheDev
	adminToken model.User
	adminAuth  model.Auth
	document   string
)

type suite struct {
	name  string
	cache Volatilizer
}

func TestMain(m *testing.M) {
	config.Current = config.LoadConfig()
	logz := logger.Get(config.Current)
	redisCache = NewCache(logz)
	devCache = NewDevCache(logz)

	adminAuth = model.Auth{
		AccountID: "047cfe5b-b91d-4ec6-9bc2-8f68309d8532",
		UserID:    "5dc37900-2a2e-46d9-8a5d-6699376975ad",
		Email:     "test@email.com",
		Role:      100,
		Token:     adminToken.Token,
	}
	document = `{"accountId":"047cfe5b-b91d-4ec6-9bc2-8f68309d8532","created":"2022-08-31T19:07:36.296787226+03:00","done":true,"id":"872c8192-c610-4fc5-bc8e-22c01f01c798","likes":0,"title":"updated","todos":[{"done":true,"title":"updated"},{"done":false,"title":"sub2"}]}`
	os.Exit(m.Run())
}

func TestCacheSubscribe(t *testing.T) {
	tests := []suite{
		{name: "subscribe with redis cache", cache: redisCache},
		{name: "subscribe with dev mem cache", cache: devCache},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			receiver := make(chan model.Command)
			closeCn := make(chan bool)
			defer close(receiver)
			defer close(closeCn)

			payload := model.Command{Type: model.MsgTypeError, Data: "invalid token", Channel: "random_cahn"}

			go tc.cache.Subscribe(receiver, "", payload.Channel, closeCn)
			time.Sleep(10 * time.Millisecond) // need to wait for proper subscriber startup

			err := tc.cache.Publish(payload)
			if err != nil {
				t.Fatal(err.Error())
			}
			timer := time.NewTimer(5 * time.Second)
			select {
			case res := <-receiver:
				if res != payload {
					t.Error("Incorrect message is received")
				}
				break
			case <-timer.C:
				t.Fatal("The channel does not received a message")

			}
			closeCn <- true
		})
	}
}

func TestCacheSubscribeOnDBEvent(t *testing.T) {
	tests := []suite{
		{name: "receive db event with redis cache", cache: redisCache},
		{name: "receive db event with dev mem cache", cache: devCache},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			receiver := make(chan model.Command)
			closeCn := make(chan bool)
			defer close(receiver)
			defer close(closeCn)

			err := tc.cache.SetTyped("token", adminAuth)
			if err != nil {
				t.Fatal(err.Error())
			}

			payload := model.Command{Type: model.MsgTypeDBUpdated, Data: document, Channel: "random_cahn"}

			go tc.cache.Subscribe(receiver, "token", payload.Channel, closeCn)

			time.Sleep(10 * time.Millisecond) // need to wait for proper subscriber startup

			err = tc.cache.Publish(payload)
			if err != nil {
				t.Fatal(err.Error())
			}
			timer := time.NewTimer(5 * time.Second)
			defer timer.Stop()
			select {
			case res := <-receiver:
				if res != payload {
					t.Error("Incorrect message is received")
				}
				break
			case <-timer.C:
				t.Fatal("The channel does not received a message")

			}
			closeCn <- true
		})
	}
}

func TestCacheDontReceiveDBEvent(t *testing.T) {
	tests := []suite{
		{
			name:  "DB event with incorrect token is not send to subscriber with redis cache",
			cache: redisCache,
		},
		{
			name:  "DB event with incorrect token is not send to subscriber with dev cache",
			cache: devCache,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			receiver := make(chan model.Command)
			closeCn := make(chan bool)
			defer close(receiver)
			defer close(closeCn)

			wrongAuth := adminAuth
			wrongAuth.AccountID = "wrongId"
			err := tc.cache.SetTyped("token", wrongAuth)
			if err != nil {
				t.Fatal(err.Error())
			}
			event := model.Command{Type: model.MsgTypeDBUpdated, Data: document, Channel: "chan"}
			go tc.cache.Subscribe(receiver, "token", event.Channel, closeCn)

			time.Sleep(10 * time.Millisecond) // need to wait for proper subscriber startup

			err = tc.cache.Publish(event)
			if err != nil {
				t.Fatal(err.Error())
			}
			timer := time.NewTimer(2 * time.Second)
			defer timer.Stop()
			select {
			case res := <-receiver:
				closeCn <- true
				t.Fatalf("The message should not be received\nReceived: %#v", res)
			case <-timer.C:
				closeCn <- true

			}
		})
	}
}

func TestCachePublishDocument(t *testing.T) {
	tests := []suite{
		{name: "receive db event with redis cache", cache: redisCache},
		{name: "receive db event with dev mem cache", cache: devCache},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			receiver := make(chan model.Command)
			closeCn := make(chan bool)
			defer close(receiver)
			defer close(closeCn)

			err := tc.cache.SetTyped("token", adminAuth)
			if err != nil {
				t.Fatal(err.Error())
			}

			payload := model.Command{Type: model.MsgTypeDBUpdated, Data: document, Channel: "random_cahn"}

			// convert to map for simulation of real usage
			var documentMap map[string]interface{}
			if err := json.Unmarshal([]byte(document), &documentMap); err != nil {
				t.Fatal(err.Error())
			}

			go tc.cache.Subscribe(receiver, "token", payload.Channel, closeCn)

			time.Sleep(10 * time.Millisecond) // need to wait for proper subscriber startup

			tc.cache.PublishDocument(payload.Channel, payload.Type, documentMap)
			timer := time.NewTimer(5 * time.Second)
			defer timer.Stop()
			select {
			case res := <-receiver:
				if res != payload {
					t.Error("Incorrect message is received")
				}
				break
			case <-timer.C:
				t.Fatal("The channel does not received a message")

			}
			closeCn <- true
		})
	}
}
