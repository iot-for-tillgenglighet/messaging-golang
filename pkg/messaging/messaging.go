package messaging

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type IoTHubMessageOrigin struct {
	Device    string  `json:"device"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type IoTHubMessage struct {
	Origin    IoTHubMessageOrigin `json:"origin"`
	Timestamp string              `json:"timestamp"`
}

// Config is an encapsulating context that wraps configuration information
// used by the initialization methods of the messaging library
type Config struct {
	ServiceName string
	Host        string
	User        string
	Password    string
}

// Context encapsulates the underlying messaging primitives, as well as
// their associated configuration
type Context interface {
	NoteToSelf(command CommandMessage) error
	SendCommandTo(command CommandMessage, key string) error
	SendResponseTo(response CommandMessage, key string) error
	PublishOnTopic(message TopicMessage) error
	Close()
	RegisterCommandHandler(contentType string, handler CommandHandler) error
	RegisterTopicMessageHandler(routingKey string, handler TopicMessageHandler)
}

type rabbitMQContext struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	cfg        Config

	commandHandlers   map[string]CommandHandler
	responseQueueName string

	connectionClosedError chan *amqp.Error
}

// CommandMessage is an interface used when sending commands
type CommandMessage interface {
	ContentType() string
}

// CommandMessageWrapper is used to wrap an incoming command message
type CommandMessageWrapper interface {
	Body() []byte
	RespondWith(CommandMessage) error
}

// NoteToSelf enqueues a command to the same routing key as the calling service
// which means that the sender or one of its replicas will receive the command
func (ctx *rabbitMQContext) NoteToSelf(command CommandMessage) error {
	return ctx.SendCommandTo(command, ctx.serviceName())
}

// SendCommandTo enqueues a command to given routing key via the command exchange
func (ctx *rabbitMQContext) SendCommandTo(command CommandMessage, key string) error {
	messageBytes, err := json.MarshalIndent(command, "", " ")
	if err != nil {
		return &Error{"Unable to marshal command to json!", err}
	}

	err = ctx.channel.Publish(commandExchange, key, true, false, amqp.Publishing{
		ContentType: command.ContentType(),
		ReplyTo:     ctx.responseQueueName,
		Body:        messageBytes,
	})
	if err != nil {
		return &Error{"Failed to publish a command to " + key + "!", err}
	}

	return nil
}

// SendResponseTo enqueues a response to a given routing key via the command exchange
func (ctx *rabbitMQContext) SendResponseTo(response CommandMessage, key string) error {
	messageBytes, err := json.MarshalIndent(response, "", " ")
	if err != nil {
		return &Error{"Unable to marshal response to json!", err}
	}

	err = ctx.channel.Publish(commandExchange, key, true, false, amqp.Publishing{
		ContentType: response.ContentType(),
		Body:        messageBytes,
	})
	if err != nil {
		return &Error{"Failed to publish a command to " + key + "!", err}
	}

	return nil
}

// TopicMessage is an interface used when sending messages to make sure
// that messages are sent to the correct topic with correct content type
type TopicMessage interface {
	ContentType() string
	TopicName() string
}

// PublishOnTopic takes a TopicMessage, reads its TopicName property,
// and publishes it to the correct topic with the correct content type
func (ctx *rabbitMQContext) PublishOnTopic(message TopicMessage) error {
	messageBytes, err := json.MarshalIndent(message, "", " ")
	if err != nil {
		return &Error{"Unable to marshal telemetry message to json!", err}
	}

	err = ctx.channel.Publish(topicExchange, message.TopicName(), false, false,
		amqp.Publishing{
			ContentType: message.ContentType(),
			Body:        messageBytes,
		})

	return err
}

// Close is a wrapper method to close both the underlying AMQP
// connection as well as the channel
func (ctx *rabbitMQContext) Close() {
	ctx.channel.Close()
	ctx.connection.Close()
}

//CommandHandler is a callback type to be used for dispatching incoming commands
type CommandHandler func(CommandMessageWrapper) error

//RegisterCommandHandler registers a handler to be called when a command with a given
//content type is received
func (ctx *rabbitMQContext) RegisterCommandHandler(contentType string, handler CommandHandler) error {
	//TODO: Return an error if a handler has already been registered
	//TODO: Mutex protection
	if ctx.commandHandlers == nil {
		ctx.commandHandlers = map[string]CommandHandler{}
	}

	ctx.commandHandlers[contentType] = handler
	return nil
}

func (ctx *rabbitMQContext) serviceName() string {
	return ctx.cfg.ServiceName
}

// Error encapsulates a lower level error together with an error
// message provided by the caller that experienced the error
type Error struct {
	msg string
	err error
}

func (err *Error) Error() string {
	if err.err != nil {
		return err.msg + " (" + err.err.Error() + ")"
	}

	return err.msg
}

const (
	commandExchange = "iot-cmd-exchange-direct"
	topicExchange   = "iot-msg-exchange-topic"
)

func getEnvironmentVariableOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// LoadConfiguration loads configuration values from RABBITMQ_HOST, RABBITMQ_USER
// and RABBITMQ_PASS. The username and password defaults to the bitnami ootb
// values for local testing.
func LoadConfiguration(serviceName string) Config {
	rabbitMQHostEnvVar := "RABBITMQ_HOST"
	rabbitMQHost := os.Getenv(rabbitMQHostEnvVar)
	rabbitMQUser := getEnvironmentVariableOrDefault("RABBITMQ_USER", "user")
	rabbitMQPass := getEnvironmentVariableOrDefault("RABBITMQ_PASS", "bitnami")
	rabbitMQDisabled := getEnvironmentVariableOrDefault("RABBITMQ_DISABLED", "false")

	if rabbitMQDisabled != "true" {
		if rabbitMQHost == "" {
			log.Fatal("Rabbit MQ host missing. Please set " + rabbitMQHostEnvVar + " to a valid host name or IP.")
		}

		return Config{
			ServiceName: serviceName,
			Host:        rabbitMQHost,
			User:        rabbitMQUser,
			Password:    rabbitMQPass,
		}
	}

	return Config{
		ServiceName: serviceName,
		Host:        "",
		User:        "",
		Password:    "",
	}
}

// Initialize takes a Config parameter and initializes a connection,
// channel, topic exchange, command exchange and service specific
// command and response queues. Retries every 2 seconds until successfull.
func Initialize(cfg Config) (Context, error) {

	if cfg.Host == "" {
		log.Info("Host name empty, returning mocked context instead.")
		return &mockedContext{}, nil
	}

	var connClosedError = make(chan *amqp.Error)
	var context *rabbitMQContext
	var err error

	for {

		time.Sleep(2 * time.Second)

		context, err = createMessageQueueChannel(&rabbitMQContext{
			cfg:                   cfg,
			connectionClosedError: connClosedError,
		})
		if err != nil {
			log.Error(err)
			continue
		}

		err = createTopicExchange(context)
		if err != nil {
			log.Error(err)
			continue
		}

		err = createCommandAndResponseQueues(context)
		if err != nil {
			log.Error(err)
			continue
		}

		return context, nil
	}
}

// TopicMessageHandler is a callback type that should be passed
// to RegisterTopicMessageHandler to receive messages from topics.
type TopicMessageHandler func(amqp.Delivery)

// RegisterTopicMessageHandler creates a subscription queue that is bound
// to the topic exchange with the provided routing key, starts a consumer
// for that queue and hands off any received messages to the provided
// TopicMessageHandler
func (ctx *rabbitMQContext) RegisterTopicMessageHandler(routingKey string, handler TopicMessageHandler) {
	queue, err := ctx.channel.QueueDeclare(
		"",    //name
		false, //durable
		false, //delete when unused
		false, //exclusive
		false, //no-wait
		nil,   //arguments
	)
	if err != nil {
		log.Fatal("Failed to declare a queue: " + err.Error())
	}
	log.Infof("Declared topic subscription queue '%s'.", queue.Name)

	err = ctx.channel.QueueBind(
		queue.Name,
		routingKey,
		topicExchange,
		false,
		nil,
	)
	if err != nil {
		log.Fatal("Failed to bind a queue: " + err.Error())
	}
	log.Infof("Successfully bound to queue '%s'.", queue.Name)

	messagesFromQueue, err := ctx.channel.Consume(
		queue.Name, //queue
		"",         //consumer
		true,       //auto ack
		true,       //exclusive
		false,      //no local
		false,      //no-wait
		nil,        //args
	)
	if err != nil {
		log.Fatal("Failed to register a consumer: " + err.Error())
	}
	log.Infof("Successfully registered as a consumer of '%s'.", queue.Name)

	go func() {
		for msg := range messagesFromQueue {
			handler(msg)
		}
	}()
}

func createMessageQueueChannel(ctx *rabbitMQContext) (*rabbitMQContext, error) {
	connectionString := fmt.Sprintf("amqp://%s:%s@%s:5672/", ctx.cfg.User, ctx.cfg.Password, ctx.cfg.Host)
	conn, err := amqp.Dial(connectionString)
	if err != nil {
		return nil, &Error{"Unable to connect to message queue!", err}
	}

	amqpChannel, err := conn.Channel()

	if err != nil {
		return nil, &Error{"Unable to create an amqp channel to message queue!", err}
	}

	ctx.connection = conn
	ctx.connection.NotifyClose(ctx.connectionClosedError)
	ctx.channel = amqpChannel

	go func() {
		for evt := range ctx.connectionClosedError {
			log.Fatal("Connection error: " + evt.Error())
		}
	}()

	return ctx, nil
}

func createCommandExchange(ctx *rabbitMQContext) error {
	err := ctx.channel.ExchangeDeclare(commandExchange, amqp.ExchangeDirect, false, false, false, false, nil)

	if err != nil {
		err = &Error{"Unable to declare command exchange " + commandExchange + "!", err}
	}

	return err
}

func createTopicExchange(ctx *rabbitMQContext) error {
	err := ctx.channel.ExchangeDeclare(topicExchange, amqp.ExchangeTopic, false, false, false, false, nil)

	if err != nil {
		err = &Error{"Unable to declare topic exchange " + topicExchange + "!", err}
	}

	return err
}

func createCommandAndResponseQueues(ctx *rabbitMQContext) error {
	err := createCommandExchange(ctx)
	if err != nil {
		return err
	}

	serviceName := ctx.serviceName()

	commandQueue, err := ctx.channel.QueueDeclare(serviceName, false, false, false, false, nil)
	if err != nil {
		return &Error{"Failed to declare command queue for " + serviceName + "!", err}
	}

	err = ctx.channel.QueueBind(commandQueue.Name, serviceName, commandExchange, false, nil)
	if err != nil {
		return &Error{"Failed to bind command queue " + commandQueue.Name + " to exchange " + commandExchange + "!", err}
	}

	responseQueue, err := ctx.channel.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		return &Error{"Failed to declare response queue for " + serviceName + "!", err}
	}

	err = ctx.channel.QueueBind(responseQueue.Name, responseQueue.Name, commandExchange, false, nil)
	if err != nil {
		msg := fmt.Sprintf("Failed to bind response queue %s to exchange %s!", responseQueue.Name, commandExchange)
		return &Error{msg, err}
	}

	commands, err := ctx.channel.Consume(commandQueue.Name, "command-consumer", false, false, false, false, nil)
	if err != nil {
		msg := fmt.Sprintf("Unable to start consuming commands from %s!", commandQueue.Name)
		return &Error{msg, err}
	}

	ctx.RegisterCommandHandler(PingCommandContentType, NewPingCommandHandler(ctx))

	go func() {
		for cmd := range commands {
			log.Info("Received command: " + string(cmd.Body))

			handler, ok := ctx.commandHandlers[cmd.ContentType]
			if ok {
				handler(newAMQPDeliveryWrapper(ctx, &cmd))
			}

			cmd.Ack(true)
		}
	}()

	responses, err := ctx.channel.Consume(responseQueue.Name, "response-consumer", false, false, false, false, nil)
	if err != nil {
		msg := fmt.Sprintf("Unable to start consuming responses from %s!", responseQueue.Name)
		return &Error{msg, err}
	}

	ctx.responseQueueName = responseQueue.Name

	err = ctx.NoteToSelf(NewPingCommand())
	if err != nil {
		return &Error{"Failed to publish a ping command to ourselves!", err}
	}

	go func() {
		for response := range responses {
			log.Info("Received response: " + string(response.Body))
			response.Ack(true)
		}
	}()

	return nil
}
