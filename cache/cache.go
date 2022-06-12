package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/internal"

	"github.com/go-redis/redis/v8"
)

type Cache struct {
	Rdb *redis.Client
	Ctx context.Context
}

// NewCache returns an initiated Redis client
func NewCache() *Cache {
	var err error
	var opt *redis.Options

	if uri := config.Current.RedisURL; len(uri) > 0 {
		opt, err = redis.ParseURL(uri)
		if err != nil {
			log.Fatal("invalid REDIS_URL value: ", err)
		}
	} else {
		opt = &redis.Options{
			Addr:     config.Current.RedisHost,
			Password: config.Current.RedisPassword,
			DB:       0, // use default DB
		}
	}
	rdb := redis.NewClient(opt)

	return &Cache{
		Rdb: rdb,
		Ctx: context.Background(),
	}
}

func (c *Cache) Get(key string) (string, error) {
	return c.Rdb.Get(c.Ctx, key).Result()
}

func (c *Cache) Set(key string, value string) error {
	if _, err := c.Rdb.Set(c.Ctx, key, value, 12*time.Hour).Result(); err != nil {
		return err
	}
	return nil
}

func (c *Cache) GetTyped(key string, v interface{}) error {
	s, err := c.Get(key)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(s), v)
}

func (c *Cache) SetTyped(key string, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Set(key, string(b))
}

func (c *Cache) Inc(key string, by int64) (int64, error) {
	return c.Rdb.IncrBy(c.Ctx, key, by).Result()
}

func (c *Cache) Dec(key string, by int64) (int64, error) {
	return c.Rdb.DecrBy(c.Ctx, key, by).Result()
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

			// TODO: this will need more thinking
			if msg.Type == internal.MsgTypeChanIn {
				msg.Type = internal.MsgTypeChanOut
			} else if msg.IsSystemEvent {

			} else if msg.IsDBEvent() && c.HasPermission(token, channel, msg.Data) == false {
				continue
			}
			send <- msg
		case <-close:
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

	// Publish the event to system so server-side function can trigger
	go func(sysmsg internal.Command) {
		sysmsg.IsSystemEvent = true
		b, err := json.Marshal(sysmsg)
		if err != nil {
			log.Println("error marshaling the system msg: ", err)
			return
		}

		sysctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		if err := c.Rdb.Publish(sysctx, "sbsys", string(b)).Err(); err != nil {
			log.Println("error publishing to system channel: ", err)
		}
	}(msg)

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
	var me internal.Auth
	if err := c.GetTyped(token, &me); err != nil {
		return false
	}

	docs := make(map[string]interface{})
	if err := json.Unmarshal([]byte(payload), &docs); err != nil {
		fmt.Println("error decoding docs for permissions check", err)
		return false
	}

	switch internal.ReadPermission(repo) {
	case internal.PermGroup:
		acctID, ok := docs["accountId"]
		if !ok {
			return false
		}

		return fmt.Sprintf("%v", acctID) == me.AccountID
	case internal.PermOwner:
		owner, ok := docs["ownerId"]
		if !ok {
			return false
		}

		return fmt.Sprintf("%v", owner) == me.UserID
	default:
		return true
	}
}

func (c *Cache) QueueWork(key, value string) error {
	return c.Rdb.RPush(c.Ctx, key, value).Err()
}

func (c *Cache) DequeueWork(key string) (string, error) {
	val, err := c.Rdb.LPop(c.Ctx, key).Result()
	if err != nil {
		if err.Error() == redis.Nil.Error() {
			return "", nil
		}
		return "", err
	}

	return val, nil
}
