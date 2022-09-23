// Package cache handles caching and pub/sub.
package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/model"

	"github.com/go-redis/redis/v8"
)

// Cache uses Redis to implement the Volatilizer interface
type Cache struct {
	Rdb *redis.Client
	Ctx context.Context
	log *logger.Logger
}

// NewCache returns an initiated Redis client
func NewCache(log *logger.Logger) *Cache {
	var err error
	var opt *redis.Options

	if uri := config.Current.RedisURL; len(uri) > 0 {
		opt, err = redis.ParseURL(uri)
		if err != nil {
			log.Fatal().Err(err).Msg("invalid REDIS_URL value")
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
		log: log,
	}
}

// Get gets a value by its id
func (c *Cache) Get(key string) (string, error) {
	return c.Rdb.Get(c.Ctx, key).Result()
}

// Set sets a value for a key
func (c *Cache) Set(key string, value string) error {
	if _, err := c.Rdb.Set(c.Ctx, key, value, 12*time.Hour).Result(); err != nil {
		return err
	}
	return nil
}

// GetTyped retrives the value for a key and unmarshal the JSON value into the
// interface
func (c *Cache) GetTyped(key string, v interface{}) error {
	s, err := c.Get(key)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(s), v)
}

// SetTyped converts the interface into JSON before storing its string value
func (c *Cache) SetTyped(key string, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Set(key, string(b))
}

// Inc increments a value (atomic inc per Redis)
func (c *Cache) Inc(key string, by int64) (int64, error) {
	return c.Rdb.IncrBy(c.Ctx, key, by).Result()
}

// Dec decreases a value (atomic per Redis)
func (c *Cache) Dec(key string, by int64) (int64, error) {
	return c.Rdb.DecrBy(c.Ctx, key, by).Result()
}

// Subscribe subscribes to a topic to receive messages on system/user events
func (c *Cache) Subscribe(send chan model.Command, token, channel string, close chan bool) {
	pubsub := c.Rdb.Subscribe(c.Ctx, channel)

	if _, err := pubsub.Receive(c.Ctx); err != nil {
		c.log.Error().Err(err).Msg("error establishing PubSub subscription")
		return
	}

	ch := pubsub.Channel()

	for {
		select {
		case m := <-ch:
			var msg model.Command
			if err := json.Unmarshal([]byte(m.Payload), &msg); err != nil {
				c.log.Error().Err(err).Msg("error parsing JSON message")
				_ = pubsub.Close()
				return
			}

			// TODO: this will need more thinking
			if msg.Type == model.MsgTypeChanIn {
				msg.Type = model.MsgTypeChanOut
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

// Publish sends a message and all subscribers will receive it if they're
// subscribed to that topic
func (c *Cache) Publish(msg model.Command) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Publish the event to system so server-side function can trigger
	go func(sysmsg model.Command) {
		sysmsg.IsSystemEvent = true
		b, err := json.Marshal(sysmsg)
		if err != nil {
			c.log.Error().Err(err).Msg("error marshaling the system msg")
			return
		}

		sysctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		if err := c.Rdb.Publish(sysctx, "sbsys", string(b)).Err(); err != nil {
			c.log.Error().Err(err).Msg("error publishing to system channel")
		}
	}(msg)

	return c.Rdb.Publish(ctx, msg.Channel, string(b)).Err()
}

// PublishDocument publishes a database update message (created, updated, deleted)
// All subscribers will get notified
func (c *Cache) PublishDocument(channel, typ string, v interface{}) {
	subs, err := c.Rdb.PubSubNumSub(c.Ctx, channel).Result()
	if err != nil {
		c.log.Error().Err(err).Msgf("error getting db subscribers for %s", channel)
		return
	}

	count, ok := subs[channel]
	if !ok {
		c.log.Warn().Msgf("cannot find channel in subs: %s", channel)
		return
	} else if count == 0 {
		return
	}

	b, err := json.Marshal(v)
	if err != nil {
		c.log.Error().Err(err).Msg("error publishing db doc")
		return
	}

	msg := model.Command{
		Channel: channel,
		Data:    string(b),
		Type:    typ,
	}

	if err := c.Publish(msg); err != nil {
		c.log.Error().Err(err).Msg("unable to publish db doc events")
	}
}

// HasPermission determines if a session token has permission to a collection
func (c *Cache) HasPermission(token, repo, payload string) bool {
	var me model.Auth
	if err := c.GetTyped(token, &me); err != nil {
		return false
	}

	docs := make(map[string]interface{})
	if err := json.Unmarshal([]byte(payload), &docs); err != nil {
		c.log.Error().Err(err).Msg("error decoding docs for permissions check")

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

// QueueWork uses Redis's LIST (atomic) as a work queue
func (c *Cache) QueueWork(key, value string) error {
	return c.Rdb.RPush(c.Ctx, key, value).Err()
}

// DequeueWork uses Redis's LIST (atomic) to get the next work queue value
// You'd typically call this from a time.Ticker for instance or in some
// kind of loop
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
