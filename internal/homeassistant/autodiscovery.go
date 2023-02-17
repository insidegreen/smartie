package homeassistant

import (
	"encoding/json"
	"fmt"
	"smarties/internal/smartie"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

// https://www.home-assistant.io/integrations/mqtt/#discovery-payload
type autoDiscovery struct {
	SensorName    string `json:"name"`
	SensorId      string `json:"unique_id"`
	DeviceClass   string `json:"device_class"`
	StateTopic    string `json:"state_topic"`
	CommandTopic  string `json:"command_topic"`
	ValueTemplate string `json:"value_template"`
	Device        device `json:"device"`
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
	msg := "on"
	if !smartieDevice.IsAcPowered {
		msg = "off"
	}

	natsConn.Publish(subj, []byte(msg))
}

func sendAutoDiscoverMessage(smartieDevice *smartie.BatteryPoweredDevice) {

	adSW := autoDiscovery{
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
	msg, err := json.Marshal(adSW)

	if err == nil {
		natsConn.Publish(subj, msg)
	} else {
		logrus.Error(err)
	}

	adML := autoDiscovery{
		SensorName:    "maintain level",
		SensorId:      "maintain_level_" + smartieDevice.NodeName,
		DeviceClass:   "battery",
		StateTopic:    "smartie/laptop/" + smartieDevice.NodeName + "/status",
		CommandTopic:  "smartie/laptop/" + smartieDevice.NodeName + "/maintain",
		ValueTemplate: "{{value_json.maintainLevel}}",
		Device: device{
			Name:        smartieDevice.NodeName,
			Identifiers: []string{smartieDevice.NodeName},
		},
	}

	subj = fmt.Sprintf("homeassistant.number.%s.config", smartieDevice.NodeName)
	msg, err = json.Marshal(adML)

	if err == nil {
		natsConn.Publish(subj, msg)
	} else {
		logrus.Error(err)
	}

}
