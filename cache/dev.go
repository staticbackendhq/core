package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/staticbackendhq/core/cache/observer"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/model"
)

// CacheDev used in local dev mode and is memory-based
type CacheDev struct {
	data     map[string]string
	log      *logger.Logger
	observer observer.Observer
	m        *sync.RWMutex
}

// NewDevCache returns a memory-based Volatilizer
func NewDevCache(log *logger.Logger) *CacheDev {
	return &CacheDev{
		data:     make(map[string]string),
		observer: observer.NewObserver(log),
		log:      log,
		m:        &sync.RWMutex{},
	}
}

// Get gets a value by its id
func (d *CacheDev) Get(key string) (val string, err error) {
	d.m.RLock()
	defer d.m.RUnlock()

	val, ok := d.data[key]
	if !ok {
		err = errors.New("key not found in cache")
	}
	return
}

// Set sets a value for a key
func (d *CacheDev) Set(key string, value string) error {
	d.m.Lock()
	defer d.m.Unlock()

	d.data[key] = value
	return nil
}

// GetTyped retrives the value for a key and unmarshal the JSON value into the
func (d *CacheDev) GetTyped(key string, v any) error {
	val, err := d.Get(key)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), v)
}

// SetTyped converts the interface into JSON before storing its string value
func (d *CacheDev) SetTyped(key string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return d.Set(key, string(b))
}

// Inc increments a value (non-atomic)
func (d *CacheDev) Inc(key string, by int64) (n int64, err error) {
	if err = d.GetTyped(key, &n); err != nil {
		return
	}

	n += by

	err = d.SetTyped(key, n)
	return
}

// Dec decrements a value (non-atomic)
func (d *CacheDev) Dec(key string, by int64) (int64, error) {
	return d.Inc(key, -1*by)
}

// Subscribe subscribes to a topic to receive messages on system/user events
func (d *CacheDev) Subscribe(send chan model.Command, token, channel string, close chan bool) {
	pubsub := d.observer.Subscribe(channel)

	ch := pubsub.Channel()

	for {
		select {
		case m := <-ch:
			var msg model.Command
			if err := json.Unmarshal([]byte(m.(string)), &msg); err != nil {
				d.log.Error().Err(err).Msg("error parsing JSON message")
				_ = pubsub.Close()
				_ = d.observer.Unsubscribe(channel, pubsub)
				return
			}

			// TODO: this will need more thinking
			if msg.Type == model.MsgTypeChanIn {
				msg.Type = model.MsgTypeChanOut
			} else if msg.IsSystemEvent {

			} else if msg.IsDBEvent() && !d.HasPermission(token, channel, msg.Data) {
				continue
			}
			send <- msg
		case <-close:
			_ = pubsub.Close()
			_ = d.observer.Unsubscribe(channel, pubsub)
			return
		}
	}
}

// Publish sends a message and all subscribers will receive it if they're
// subscribed to that topic
func (d *CacheDev) Publish(msg model.Command) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Publish the event to system so server-side function can trigger
	// but only for non system msg
	if !msg.IsSystemEvent && msg.Channel != "sbsys" {
		go func(sysmsg model.Command) {
			sysmsg.IsSystemEvent = true
			b, err := json.Marshal(sysmsg)
			if err != nil {
				d.log.Error().Err(err).Msg("error marshaling the system msg")
				return
			}
			if err := d.observer.Publish("sbsys", string(b)); err != nil {
				d.log.Error().Err(err).Msg("error occurred during publishing to 'sbsys' channel")
			}
		}(msg)
	}

	subs := d.observer.PubNumSub(msg.Channel)

	count, ok := subs[msg.Channel]
	if !ok {
		d.log.Warn().Msgf("cannot find channel in subs: %s", msg.Channel)
		return nil
	} else if count == 0 {
		return nil
	}
	return d.observer.Publish(msg.Channel, string(b))
}

// PublishDocument publishes a database update message (created, updated, deleted)
// All subscribers will get notified
func (d *CacheDev) PublishDocument(auth model.Auth, dbName, channel, typ string, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		d.log.Error().Err(err).Msg("error publishing db doc")
		return
	}

	msg := model.Command{
		Channel: channel,
		Data:    string(b),
		Type:    typ,
		Auth:    auth,
		Base:    dbName,
	}

	if err := d.Publish(msg); err != nil {
		d.log.Error().Err(err).Msg("unable to publish db doc events")
	}
}

// HasPermission determines if a session token has permission to a collection
func (d *CacheDev) HasPermission(token, repo, payload string) bool {
	if repo == "sbsys" {
		return true
	}

	var me model.Auth
	if err := d.GetTyped(token, &me); err != nil {
		return false
	}

	docs := make(map[string]interface{})
	if err := json.Unmarshal([]byte(payload), &docs); err != nil {
		d.log.Error().Err(err).Msg("error decoding docs for permissions check")

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

// QueueWork uses a slice to replicate a work queue (non-atomic)
func (d *CacheDev) QueueWork(key, value string) error {
	var queue []string
	if err := d.GetTyped(key, &queue); err != nil {
		queue = make([]string, 0)
	}

	queue = append(queue, value)

	return d.SetTyped(key, queue)
}

// DequeueWork uses a string slice to replicate a work queue (non-atomic)
// You'd typically call this from a time.Ticker for instance or in some
// kind of loop
func (d *CacheDev) DequeueWork(key string) (val string, err error) {
	var queue []string
	if err = d.GetTyped(key, &queue); err != nil {
		return
	} else if len(queue) == 0 {
		return
	}

	val = queue[0]

	err = d.SetTyped(key, queue[1:])
	return
}
