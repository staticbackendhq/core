package cache

import "github.com/staticbackendhq/core/model"

// PublishDocumentEvent used to publish database events
type PublishDocumentEvent func(channel, typ string, v interface{})

// Volatilizer is the cache and pub/sub interface
type Volatilizer interface {
	// Get returns a string value from a key
	Get(key string) (string, error)
	// Set sets a string value
	Set(key string, value string) error
	// GetTyped returns a typed struct by its key
	GetTyped(key string, v any) error
	// SetTyped sets a typed struct for a key
	SetTyped(key string, v any) error
	// Inc increments a numeric value for a key
	Inc(key string, by int64) (int64, error)
	// Dec decrements a value for a key
	Dec(key string, by int64) (int64, error)
	// Subscribe subscribes to a pub/sub channel
	Subscribe(send chan model.Command, token, channel string, close chan bool)
	// Publish publishes a message to a channel
	Publish(msg model.Command) error
	// PublishDocument publish a database message to a channel
	PublishDocument(channel, typ string, v any)
	// QueueWork add a work queue item
	QueueWork(key, value string) error
	// DequeueWork dequeue work item (if available)
	DequeueWork(key string) (string, error)
}
