package main

import (
	"context"
	"encoding/json"
	"log"

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

			msg.Type = MsgTypeChanOut
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
	return c.Rdb.Publish(c.Ctx, msg.Channel, string(b)).Err()
}
