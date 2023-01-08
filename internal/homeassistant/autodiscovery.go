package homeassistant

import (
	"encoding/json"
	"fmt"
	"smarties/internal/smartie"

	"github.com/nats-io/nats.go"
)

// https://www.home-assistant.io/integrations/mqtt/#discovery-payload
type autoDiscovery struct {
	SensorName   string `json:"name"`
	SensorId     string `json:"unique_id"`
	DeviceClass  string `json:"device_class"`
	StateTopic   string `json:"state_topic"`
	CommandTopic string `json:"command_topic"`
	Device       device `json:"device"`
}

type device struct {
	Name        string   `json:"name"`
	Identifiers []string `json:"identifiers"`
}

var natsConn *nats.Conn

func Operate(nc *nats.Conn) {
	natsConn = nc
	nc.Subscribe("smartie.laptop.*.status", haHandler)
}

func haHandler(m *nats.Msg) {
	bpd := &smartie.BatteryPoweredDevice{}
	json.Unmarshal(m.Data, bpd)
	go sendAutoDiscoverMessage(bpd)
	go sendStateMessage(bpd)
}

func sendStateMessage(smartieDevice *smartie.BatteryPoweredDevice) {

	subj := fmt.Sprintf("smartie.laptop.%s.state", smartieDevice.NodeName)
	msg := "ON"
	if !smartieDevice.IsAcPowered {
		msg = "OFF"
	}

	natsConn.Publish(subj, []byte(msg))
}

func sendAutoDiscoverMessage(smartieDevice *smartie.BatteryPoweredDevice) {

	ad := autoDiscovery{
		SensorName:   "power switch",
		SensorId:     "power_switch_" + smartieDevice.NodeName,
		DeviceClass:  "switch",
		StateTopic:   "smartie/laptop/" + smartieDevice.NodeName + "/state",
		CommandTopic: "smartie/laptop/" + smartieDevice.NodeName + "/plug",
		Device: device{
			Name:        smartieDevice.NodeName,
			Identifiers: []string{smartieDevice.NodeName},
		},
	}

	subj := fmt.Sprintf("homeassistant.switch.%s.config", smartieDevice.NodeName)
	msg, err := json.Marshal(ad)

	if err == nil {
		natsConn.Publish(subj, msg)
	}
}
