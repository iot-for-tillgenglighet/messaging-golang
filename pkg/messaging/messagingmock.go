package messaging

type mockedContext struct {
}

// NoteToSelf enqueues a command to the same routing key as the calling service
// which means that the sender or one of its replicas will receive the command
func (ctx *mockedContext) NoteToSelf(command CommandMessage) error {
	return nil
}

// SendCommandTo enqueues a command to given routing key via the command exchange
func (ctx *mockedContext) SendCommandTo(command CommandMessage, key string) error {
	return nil
}

// SendResponseTo enqueues a response to a given routing key via the command exchange
func (ctx *mockedContext) SendResponseTo(response CommandMessage, key string) error {
	return nil
}

// PublishOnTopic takes a TopicMessage, reads its TopicName property,
// and publishes it to the correct topic with the correct content type
func (ctx *mockedContext) PublishOnTopic(message TopicMessage) error {
	return nil
}

// Close is a wrapper method to close both the underlying AMQP
// connection as well as the channel
func (ctx *mockedContext) Close() {

}

//RegisterCommandHandler registers a handler to be called when a command with a given
//content type is received
func (ctx *mockedContext) RegisterCommandHandler(contentType string, handler CommandHandler) error {

	return nil
}

// RegisterTopicMessageHandler creates a subscription queue that is bound
// to the topic exchange with the provided routing key, starts a consumer
// for that queue and hands off any received messages to the provided
// TopicMessageHandler
func (ctx *mockedContext) RegisterTopicMessageHandler(routingKey string, handler TopicMessageHandler) {

}
