package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	return &Cache{
		Rdb: rdb,
		Ctx: context.Background(),
	}
}

func (c *Cache) Subscribe(send chan Command, channel string, close chan bool) {
	pubsub := c.Rdb.Subscribe(c.Ctx, channel)

	if _, err := pubsub.Receive(c.Ctx); err != nil {
		log.Println("error establishing PubSub subscription", err)
		return
	}

	ch := pubsub.Channel()

	for {
		select {
		case m := <-ch:
			var msg Command
			if err := json.Unmarshal([]byte(m.Payload), &msg); err != nil {
				log.Println("error parsing JSON message", err)
				_ = pubsub.Close()
				return
			}

			// for non DB events we change the type to MsgTypeChanOut
			if !msg.IsDBEvent() {
				msg.Type = MsgTypeChanOut
			}
			send <- msg
		case _ = <-close:
			_ = pubsub.Close()
			return
		}
	}
}

func (c *Cache) Publish(msg Command) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	return c.Rdb.Publish(ctx, msg.Channel, string(b)).Err()
}

func (c *Cache) publishDocument(channel, typ string, v interface{}) {
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

	msg := Command{
		Channel: channel,
		Data:    string(b),
		Type:    typ,
	}

	if err := c.Publish(msg); err != nil {
		fmt.Println("unable to publish db doc events:", err)
	}
}
