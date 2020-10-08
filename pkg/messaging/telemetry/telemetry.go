package telemetry

import "github.com/iot-for-tillgenglighet/messaging-golang/pkg/messaging"

// Temperature is a telemetry type IoTHubMessage
type Temperature struct {
	messaging.IoTHubMessage
	Temp float64 `json:"temp"`
}

// ContentType returns the ContentType for a Temperature telemetry message
func (msg *Temperature) ContentType() string {
	return "application/json"
}

// TopicName returns the name of the topic that a Temperature telemetry message should be posted to
func (msg *Temperature) TopicName() string {
	return "telemetry.temperature"
}

// WaterTemperature is a telemetry type IoTHubMessage
type WaterTemperature struct {
	messaging.IoTHubMessage
	Temp float64 `json:"temp"`
}

// ContentType returns the ContentType for a WaterTemperature telemetry message
func (msg *WaterTemperature) ContentType() string {
	return "application/json"
}

// TopicName returns the name of the topic that a WaterTemperature telemetry message should be posted to
func (msg *WaterTemperature) TopicName() string {
	return "telemetry.watertemperature"
}

// Problem contains information about a certain problem (only type for now)
type Problem struct {
	Type string `json:"type"`
}

// ProblemReport is a telemetry type IoTHubMessage
type ProblemReport struct {
	messaging.IoTHubMessage
	Problem Problem `json:"problem"`
}

// ContentType returns the ContentType for a ProblemReport telemetry message
func (msg *ProblemReport) ContentType() string {
	return "application/json"
}

// TopicName returns the correct topic name for a ProblemReport telemetry message
func (msg *ProblemReport) TopicName() string {
	return "telemetry.problemreport"
}

// Snowdepth is a telemetry type IoTHubMessage
type Snowdepth struct {
	messaging.IoTHubMessage
	Depth float32 `json:"depth"`
}

// ContentType returns the ContentType for a Snowdepth telemetry message
func (msg *Snowdepth) ContentType() string {
	return "application/json"
}

// TopicName returns the correct topic name for a Snowdepth telemetry message
func (msg *Snowdepth) TopicName() string {
	return "telemetry.snowdepth"
}
