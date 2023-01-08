package smartie

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

type BatteryPoweredDevice struct {
	NodeName      string `json:"nodeName"`
	BatteryLevel  int    `json:"batteryLevel"`
	IsAcPowered   bool   `json:"isAcPowered"`
	IsCharging    bool   `json:"isCharging"`
	IsLaptop      bool   `json:"isLaptop"`
	MaintainLevel int    `json:"maintainLevel"`
	// MinMaintainLevel   int    `json:"minMaintainLevel"`
	// MaxMaintainLevel   int    `json:"maxMaintainLevel"`
	SmartChargeEnabled bool `json:"smartChargeEnabled"`
}

var battDeviceMap map[string]*BatteryPoweredDevice = make(map[string]*BatteryPoweredDevice)

func updateBatteryPoweredDevice(m *nats.Msg) {

	bpd := &BatteryPoweredDevice{}
	json.Unmarshal(m.Data, bpd)

	currentBpd, exists := battDeviceMap[bpd.NodeName]

	acSwitch := false

	if !exists {
		currentBpd = bpd
		battDeviceMap[bpd.NodeName] = currentBpd
	} else {
		acSwitch = bpd.IsAcPowered != currentBpd.IsAcPowered
		json.Unmarshal(m.Data, battDeviceMap[bpd.NodeName])
	}

	if acSwitch {
		//Laptop AC Power Change
		laptopAcEvent <- currentBpd
	}

	// criticalBattState := currentBpd.BatteryLevel < currentBpd.MaintainLevel

	if !currentBpd.IsAcPowered && currentBpd.BatteryLevel <= currentBpd.MaintainLevel {
		for _, plug := range plugDeviceMap {
			if plug.pluggedDevice != nil && plug.pluggedDevice == currentBpd {
				plug.setPlugStatus("on", natsConn)
				return
			}
		}
		logrus.Info("could not find plug ... laptop is plugged somewhere")
	}
	// else if currentBpd.IsAcPowered && criticalBattState && !currentBpd.IsCharging {
	// 	currentBpd.setBatteryChargeStatus("on", natsConn)
	// }
}

func (device *BatteryPoweredDevice) setBatteryChargeStatus(status string, nats NatsInterface) {
	subject := fmt.Sprintf("smartie.laptop.%s.charge", device.NodeName)
	msg, err := nats.Request(subject, []byte(status), time.Second*5)

	if err == nil && string(msg.Data) == "ok" {
		device.IsCharging = status == "on"
	} else if err != nil {
		logrus.Errorf("Could not get a response on subject %s - %s", subject, err.Error())
	} else if string(msg.Data) != "ok" {
		logrus.Errorf("Could not get a posititve response on subject %s:  %s", subject, string(msg.Data))
	}
}

// func (device *BatteryPoweredDevice) setBatteryMaintainLevel(level int, nats NatsInterface) {
// 	subject := fmt.Sprintf("smartie.laptop.%s.maintain", device.NodeName)
// 	msg, err := nats.Request(subject, []byte(fmt.Sprint(level)), time.Second*5)

// 	if err == nil && string(msg.Data) == "ok" {
// 		device.MaintainLevel = level
// 	} else if err != nil {
// 		logrus.Errorf("Could not get a response on subject %s - %s", subject, err.Error())
// 	} else if string(msg.Data) != "ok" {
// 		logrus.Errorf("Could not get a posititve response on subject %s:  %s", subject, string(msg.Data))
// 	}
// }
