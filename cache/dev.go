package cache

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/staticbackendhq/core/cache/observer"
	"github.com/staticbackendhq/core/internal"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/model"
)

type CacheDev struct {
	data     map[string]string
	log      *logger.Logger
	observer observer.Observer
}

func NewDevCache(log *logger.Logger) *CacheDev {
	return &CacheDev{
		data:     make(map[string]string),
		observer: observer.NewObserver(log),
		log:      log,
	}
}

func (d *CacheDev) Get(key string) (val string, err error) {
	val, ok := d.data[key]
	if !ok {
		err = errors.New("key not found in cache")
	}
	return
}

func (d *CacheDev) Set(key string, value string) error {
	d.data[key] = value
	return nil
}

func (d *CacheDev) GetTyped(key string, v any) error {
	val, err := d.Get(key)
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), v)
}

func (d *CacheDev) SetTyped(key string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return d.Set(key, string(b))
}

func (d *CacheDev) Inc(key string, by int64) (n int64, err error) {
	if err = d.GetTyped(key, &n); err != nil {
		return
	}

	n += by

	err = d.SetTyped(key, n)
	return
}

func (d *CacheDev) Dec(key string, by int64) (int64, error) {
	return d.Inc(key, -1*by)
}

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

			} else if msg.IsDBEvent() && d.HasPermission(token, channel, msg.Data) == false {
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

func (d *CacheDev) Publish(msg model.Command) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Publish the event to system so server-side function can trigger
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

	return d.observer.Publish(msg.Channel, string(b))
}

func (d *CacheDev) PublishDocument(channel, typ string, v any) {
	subs := d.observer.PubNumSub(channel)

	count, ok := subs[channel]
	if !ok {
		d.log.Warn().Msgf("cannot find channel in subs: %s", channel)
		return
	} else if count == 0 {
		return
	}

	b, err := json.Marshal(v)
	if err != nil {
		d.log.Error().Err(err).Msg("error publishing db doc")
		return
	}

	msg := model.Command{
		Channel: channel,
		Data:    string(b),
		Type:    typ,
	}

	if err := d.Publish(msg); err != nil {
		d.log.Error().Err(err).Msg("unable to publish db doc events")
	}
}

func (d *CacheDev) HasPermission(token, repo, payload string) bool {
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

func (d *CacheDev) QueueWork(key, value string) error {
	var queue []string
	if err := d.GetTyped(key, &queue); err != nil {
		queue = make([]string, 0)
	}

	queue = append(queue, value)

	return d.SetTyped(key, queue)
}

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
