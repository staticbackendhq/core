package observer

import (
	"errors"
	"github.com/staticbackendhq/core/logger"
	"sync"
	"time"
)

type Observer interface {
	Subscribe(channel string) Subscriber
	Publish(channel string, msg interface{}) error
	Unsubscribe(channel string, subscriber Subscriber) error
	PubNumSub(channel string) map[string]int
}

type Subscriber interface {
	Channel() <-chan interface{}
	Close() error
}

type memSubscriber struct {
	closed bool
	msgCh  chan interface{}
}

func NewSubscriber() *memSubscriber {
	ch := make(chan interface{})
	sub := &memSubscriber{closed: false, msgCh: ch}
	return sub
}

func (ps *memSubscriber) Channel() <-chan interface{} {
	return ps.msgCh
}

func (ps *memSubscriber) Close() error {
	if ps.closed {
		return errors.New("channel is already closed")
	}
	close(ps.msgCh)
	ps.closed = true
	return nil
}

type memObserver struct {
	Subscriptions map[string][]*memSubscriber
	mx            sync.Mutex
	log           *logger.Logger
}

func NewObserver(log *logger.Logger) Observer {
	subs := make(map[string][]*memSubscriber)
	return &memObserver{Subscriptions: subs, mx: sync.Mutex{}, log: log}
}

func (o *memObserver) Subscribe(channel string) Subscriber {
	o.mx.Lock()
	defer o.mx.Unlock()

	newSub := NewSubscriber()
	if subs, ok := o.Subscriptions[channel]; ok {
		o.Subscriptions[channel] = append(subs, newSub)
	} else if !ok {
		o.Subscriptions[channel] = append([]*memSubscriber{}, newSub)
	}
	return newSub
}

func (o *memObserver) Publish(channel string, msg interface{}) error {
	o.mx.Lock()
	defer o.mx.Unlock()
	if len(o.Subscriptions[channel]) == 0 {
		return errors.New("not subscribers for chan: " + channel)
	}
	for _, sub := range o.Subscriptions[channel] {
		sub := sub

		go func() {
			timer := time.NewTimer(15 * time.Second)
			select {
			case sub.msgCh <- msg:
				if !timer.Stop() {
					<-timer.C
				}
			case <-timer.C:
				o.log.Error().Msg("the previous message is not read; dropping this message")
				timer.Stop()
			}
		}()

	}
	return nil
}

func (o *memObserver) Unsubscribe(channel string, subscriber Subscriber) error {
	o.mx.Lock()
	defer o.mx.Unlock()
	for k, v := range o.Subscriptions[channel] {
		if v == subscriber {
			o.Subscriptions[channel] = append(o.Subscriptions[channel][:k], o.Subscriptions[channel][k+1:]...)
			return nil
		}
	}
	return errors.New("given subscriber is already unsubscribed")
}

func (o *memObserver) PubNumSub(channel string) map[string]int {
	res := make(map[string]int)
	o.mx.Lock()
	defer o.mx.Unlock()
	res[channel] = len(o.Subscriptions[channel])
	return res
}
