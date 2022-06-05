package internal

type PublishDocumentEvent func(channel, typ string, v interface{})

type Volatilizer interface {
	Get(key string) (string, error)
	Set(key string, value string) error
	GetTyped(key string, v any) error
	SetTyped(key string, v any) error
	Inc(key string, by int64) (int64, error)
	Dec(key string, by int64) (int64, error)
	Subscribe(send chan Command, token, channel string, close chan bool)
	Publish(msg Command) error
	PublishDocument(channel, typ string, v any)
	QueueWork(key, value string) error
	DequeueWork(key string) (string, error)
}
