package messaging

import (
	"encoding/json"
	"fmt"

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

type Config struct {
	ServiceName string
	Host        string
	User        string
	Password    string
}

type Context struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	cfg        Config
}

type TopicMessage interface {
	ContentType() string
	TopicName() string
}

func (ctx *Context) PublishOnTopic(message TopicMessage) error {
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

func (ctx *Context) Close() {
	ctx.channel.Close()
	ctx.connection.Close()
}

func (ctx *Context) serviceName() string {
	return ctx.cfg.ServiceName
}

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

var commandExchange = "iot-cmd-exchange-direct"
var topicExchange = "iot-msg-exchange-topic"

func Initialize(cfg Config) (*Context, error) {

	context, err := createMessageQueueChannel(&Context{cfg: cfg})
	if err != nil {
		return nil, err
	}

	err = createTopicExchange(context)
	if err != nil {
		return nil, err
	}

	err = createCommandAndResponseQueues(context)
	if err != nil {
		return nil, err
	}

	return context, nil
}

func createMessageQueueChannel(ctx *Context) (*Context, error) {
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
	ctx.channel = amqpChannel

	return ctx, nil
}

func createCommandExchange(ctx *Context) error {
	err := ctx.channel.ExchangeDeclare(commandExchange, amqp.ExchangeDirect, false, false, false, false, nil)

	if err != nil {
		err = &Error{"Unable to declare command exchange " + commandExchange + "!", err}
	}

	return err
}

func createTopicExchange(ctx *Context) error {
	err := ctx.channel.ExchangeDeclare(topicExchange, amqp.ExchangeTopic, false, false, false, false, nil)

	if err != nil {
		err = &Error{"Unable to declare topic exchange " + topicExchange + "!", err}
	}

	return err
}

func createCommandAndResponseQueues(ctx *Context) error {
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

	go func() {
		for cmd := range commands {
			log.Info("Received command: " + string(cmd.Body))

			err = ctx.channel.Publish(commandExchange, cmd.ReplyTo, true, false, amqp.Publishing{
				ContentType: "application/json",
				Body:        []byte("{\"cmd\": \"pong\"}"),
			})
			if err != nil {
				log.Error("Failed to publish a pong response to ourselves! : " + err.Error())
			}

			cmd.Ack(true)
		}
	}()

	responses, err := ctx.channel.Consume(responseQueue.Name, "response-consumer", false, false, false, false, nil)
	if err != nil {
		msg := fmt.Sprintf("Unable to start consuming responses from %s!", responseQueue.Name)
		return &Error{msg, err}
	}

	err = ctx.channel.Publish(commandExchange, serviceName, true, false, amqp.Publishing{
		ContentType: "application/json",
		ReplyTo:     responseQueue.Name,
		Body:        []byte("{\"cmd\": \"ping\"}"),
	})
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
