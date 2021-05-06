package messaging

import (
	"encoding/json"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	//PingCommandContentType is the content type for a ping command
	PingCommandContentType = "application/vnd-ping-command"
	//PongResponseContentType is the content type for a pong response
	PongResponseContentType = "application/vnd-pong-response"
)

//PingCommand is a utility command to check the messenger connection
type PingCommand struct {
	Cmd       string    `json:"cmd"`
	Timestamp time.Time `json:"timestamp"`
}

//ContentType returns the content type for a ping command
func (cmd PingCommand) ContentType() string {
	return PingCommandContentType
}

//NewPingCommand instantiates a new ping command
func NewPingCommand() CommandMessage {
	return PingCommand{
		Cmd:       "ping",
		Timestamp: time.Now().UTC(),
	}
}

//NewPingCommandHandler returns a callback function to be called when ping commands
//are received
func NewPingCommandHandler(ctx Context) CommandHandler {
	return func(wrapper CommandMessageWrapper) error {
		ping := &PingCommand{}
		err := json.Unmarshal(wrapper.Body(), ping)

		err = wrapper.RespondWith(NewPongResponse(*ping))
		if err != nil {
			log.Error("Failed to publish a pong response to ourselves! : " + err.Error())
		}

		return err
	}
}

//PongResponse is a utility response to check the messenger connection
type PongResponse struct {
	Cmd       string `json:"cmd"`
	PingSent  time.Time
	Timestamp time.Time `json:"timestamp"`
}

//ContentType returns the content type for a pong response
func (cmd PongResponse) ContentType() string {
	return PingCommandContentType
}

//NewPongResponse instantiates a new pong response from a ping command
func NewPongResponse(ping PingCommand) CommandMessage {
	return PongResponse{
		Cmd:       "pong",
		PingSent:  ping.Timestamp,
		Timestamp: time.Now().UTC(),
	}
}
