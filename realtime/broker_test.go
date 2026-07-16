package realtime

import (
	"context"
	"testing"
	"time"

	"github.com/staticbackendhq/core/cache"
)

func TestBrokerCloseIsIdempotent(t *testing.T) {

	b := NewBroker(func(context.Context, string) (string, error) {
		return "", nil
	}, cache.NewDevCache())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := b.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if err := b.Close(ctx); err != nil {
		t.Fatal(err)
	}
}
