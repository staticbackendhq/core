package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"staticbackend/internal"
	"time"

	"github.com/go-redis/redis/v8"
)

type Cache struct {
	Rdb *redis.Client
	Ctx context.Context
}

// NewCache returns an initiated Redis client
func NewCache() *Cache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0, // use default DB
	})

	return &Cache{
		Rdb: rdb,
		Ctx: context.Background(),
	}
}

func (c *Cache) Get(key string) (string, error) {
	return c.Rdb.Get(c.Ctx, key).Result()
}

func (c *Cache) Set(key string, value string) error {
	if _, err := c.Rdb.Set(c.Ctx, key, value, 3*time.Hour).Result(); err != nil {
		return err
	}
	return nil
}

func (c *Cache) Subscribe(send chan internal.Command, token, channel string, close chan bool) {
	pubsub := c.Rdb.Subscribe(c.Ctx, channel)

	if _, err := pubsub.Receive(c.Ctx); err != nil {
		log.Println("error establishing PubSub subscription", err)
		return
	}

	ch := pubsub.Channel()

	for {
		select {
		case m := <-ch:
			var msg internal.Command
			if err := json.Unmarshal([]byte(m.Payload), &msg); err != nil {
				log.Println("error parsing JSON message", err)
				_ = pubsub.Close()
				return
			}

			// for non DB events we change the type to MsgTypeChanOut
			if !msg.IsDBEvent() {
				msg.Type = internal.MsgTypeChanOut
			} else if c.HasPermission(token, channel, msg.Data) == false {
				continue
			}
			send <- msg
		case _ = <-close:
			_ = pubsub.Close()
			return
		}
	}
}

func (c *Cache) Publish(msg internal.Command) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	return c.Rdb.Publish(ctx, msg.Channel, string(b)).Err()
}

func (c *Cache) PublishDocument(channel, typ string, v interface{}) {
	subs, err := c.Rdb.PubSubNumSub(c.Ctx, channel).Result()
	if err != nil {
		fmt.Println("error getting db subscribers for ", channel)
		return
	}

	count, ok := subs[channel]
	if !ok {
		fmt.Println("cannot find channel in subs", channel)
		return
	} else if count == 0 {
		return
	}

	b, err := json.Marshal(v)
	if err != nil {
		fmt.Println("error publishing db doc: ", err)
		return
	}

	msg := internal.Command{
		Channel: channel,
		Data:    string(b),
		Type:    typ,
	}

	if err := c.Publish(msg); err != nil {
		fmt.Println("unable to publish db doc events:", err)
	}
}

func (c *Cache) HasPermission(token, repo, payload string) bool {
	me, ok := internal.Tokens[token]
	if !ok {
		return false
	}

	docs := make(map[string]interface{})
	if err := json.Unmarshal([]byte(payload), &docs); err != nil {
		fmt.Println("error decoding docs for permissions check", err)
		return false
	}

	switch internal.ReadPermission(repo) {
	case internal.PermGroup:
		acctID, ok := docs[internal.FieldAccountID]
		if !ok {
			return false
		}

		return fmt.Sprintf("%v", acctID) == me.AccountID.Hex()
	case internal.PermOwner:
		owner, ok := docs[internal.FieldOwnerID]
		if !ok {
			return false
		}

		return fmt.Sprintf("%v", owner) == me.UserID.Hex()
	default:
		return true
	}
}
