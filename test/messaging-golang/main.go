package main

import (
	"os"
	"time"

	"github.com/iot-for-tillgenglighet/messaging-golang/pkg/messaging"

	log "github.com/sirupsen/logrus"
)

func getEnvironmentVariableOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {

	serviceName := "messaging-golang-test"

	log.Infof("Starting up %s ...", serviceName)

	rabbitMQHostEnvVar := "RABBITMQ_HOST"
	rabbitMQHost := os.Getenv(rabbitMQHostEnvVar)
	rabbitMQUser := getEnvironmentVariableOrDefault("RABBITMQ_USER", "user")
	rabbitMQPass := getEnvironmentVariableOrDefault("RABBITMQ_PASS", "bitnami")

	if rabbitMQHost == "" {
		log.Fatal("Rabbit MQ host missing. Please set " + rabbitMQHostEnvVar + " to a valid host name or IP.")
	}

	var messenger *messaging.Context
	var err error

	for messenger == nil {

		time.Sleep(2 * time.Second)

		messenger, err = messaging.Initialize(messaging.Config{
			ServiceName: serviceName,
			Host:        rabbitMQHost,
			User:        rabbitMQUser,
			Password:    rabbitMQPass,
		})

		if err != nil {
			log.Error(err)
		}
	}

	defer messenger.Close()
}
