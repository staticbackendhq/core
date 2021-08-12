package internal

// PubSuber contains functions to make realtime communication distributed
type PubSuber interface {
	Get(key string) (string, error)
	Set(key string, value string) error
	Inc(key string, by int64) (int64, error)
	Dec(key string, by int64) (int64, error)
	Subscribe(send chan Command, token, channel string, close chan bool)
	Publish(msg Command) error
	PublishDocument(channel, typ string, v interface{})
}
