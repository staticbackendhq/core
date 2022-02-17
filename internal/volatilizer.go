package internal

type PublishDocumentEvent func(channel, typ string, v interface{})
