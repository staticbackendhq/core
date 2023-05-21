package search_test

import (
	"testing"
	"time"

	"github.com/staticbackendhq/core/cache"
	"github.com/staticbackendhq/core/config"
	"github.com/staticbackendhq/core/logger"
	"github.com/staticbackendhq/core/model"
	"github.com/staticbackendhq/core/search"
)

func TestSearchIndexAndQuery(t *testing.T) {
	c := config.AppConfig{}

	l := logger.Get(c)

	pubsub := cache.NewDevCache(l)

	go fakeSySubscriber(pubsub, l)

	s, err := search.New("testdata/test.fts", pubsub)
	if err != nil {
		t.Fatal(err)
	}

	// give some times for go routines to kick-in
	// and create the subscriptions
	time.Sleep(1250 * time.Millisecond)

	err = s.Index("test", "catalog", "123", "this is the first doc")
	if err != nil {
		t.Fatal(err)
	}

	err = s.Index("test", "catalog", "456", "this is the 2nd doc")
	if err != nil {
		t.Fatal(err)
	}

	// let time for go rountines to propagate the system
	// event to create new full-text index
	time.Sleep(4250 * time.Millisecond)

	results, err := s.Search("test", "catalog", "first doc")
	if err != nil {
		t.Fatal(err)
	} else if len(results.IDs) != 1 {
		t.Errorf("expected 1 result, got %d", len(results.IDs))
	} else if results.IDs[0] != "123" {
		t.Log(results)
		t.Errorf("expected id to be test_catalog_123 got %s", results.IDs[0])
	}
}

func fakeSySubscriber(pubsub cache.Volatilizer, l *logger.Logger) {
	receiver := make(chan model.Command)
	close := make(chan bool)

	go pubsub.Subscribe(receiver, "", "sbsys", close)

	timer := time.NewTimer(10 * time.Second)
	for {
		select {
		case msg := <-receiver:
			l.Debug().Msg("rcvd in fake sbsys subscriber: " + msg.Type)
		case <-close:
			return
		case <-timer.C:
			return
		}
	}
}
