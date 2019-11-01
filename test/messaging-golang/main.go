package main

import (
	"encoding/json"
	"time"

	"github.com/iot-for-tillgenglighet/messaging-golang/pkg/messaging"
	"github.com/iot-for-tillgenglighet/messaging-golang/pkg/messaging/telemetry"
	"github.com/streadway/amqp"

	log "github.com/sirupsen/logrus"
)

func messageHandler(message amqp.Delivery) {
	log.Info("Message received from queue: " + string(message.Body))
	msg := &telemetry.Temperature{}

	err := json.Unmarshal(message.Body, msg)
	if err != nil {
		log.Error("Failed to unmarshal message: " + err.Error())
	}
}

func main() {

	log.Info("Starting up ...")

	serviceName := "messaging-golang-test"
	config := messaging.LoadConfiguration(serviceName)

	var messenger *messaging.Context
	var err error

	for messenger == nil {

		time.Sleep(2 * time.Second)

		messenger, err = messaging.Initialize(config)

		if err != nil {
			log.Error(err)
		}
	}

	testMessage := &telemetry.Temperature{
		Temp: 37.0,
	}

	messenger.RegisterTopicMessageHandler(testMessage.TopicName(), messageHandler)
	messenger.PublishOnTopic(testMessage)

	time.Sleep(5 * time.Second)

	defer messenger.Close()
}
