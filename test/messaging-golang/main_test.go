package main

import (
	"os"
	"testing"

	"github.com/iot-for-tillgenglighet/messaging-golang/pkg/messaging"
)

func TestLoadMockConfigAndInitialize(t *testing.T) {
	os.Setenv("RABBITMQ_DISABLED", "false")
	conf := messaging.LoadConfiguration("messaging-golang-test")

	_, err := messaging.Initialize(conf)
	if err != nil {
		t.Error(err)
	}

}
