package smartie

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

type BatteryPoweredDevice struct {
	NodeName           string `json:"nodeName"`
	BatteryLevel       int    `json:"batteryLevel"`
	IsAcPowered        bool   `json:"isAcPowered"`
	IsCharging         bool   `json:"isCharging"`
	IsLaptop           bool   `json:"isLaptop"`
	MaintainLevel      int    `json:"maintainLevel"`
	MinMaintainLevel   int    `json:"minMaintainLevel"`
	MaxMaintainLevel   int    `json:"maxMaintainLevel"`
	SmartChargeEnabled bool   `json:"smartChargeEnabled"`
}

var battDeviceMap map[string]*BatteryPoweredDevice = make(map[string]*BatteryPoweredDevice)

func updateBatteryPoweredDevice(m *nats.Msg) {

	bpd := &BatteryPoweredDevice{}
	json.Unmarshal(m.Data, bpd)

	currentBpd, exists := battDeviceMap[bpd.NodeName]

	if !exists {
		currentBpd = bpd
		battDeviceMap[bpd.NodeName] = currentBpd
	}

	acSwitch := bpd.IsAcPowered != currentBpd.IsAcPowered

	battDeviceMap[bpd.NodeName] = currentBpd

	if acSwitch {
		//Laptop AC Power Change
		laptopAcEvent <- bpd
	}

	if currentBpd.IsAcPowered {
		if currentBpd.BatteryLevel <= currentBpd.MinMaintainLevel {
			currentBpd.setBatteryChargeStatus("on", natsConn)
		}
	}
	// else find and activate plug
}

func (device *BatteryPoweredDevice) setBatteryChargeStatus(status string, nats NatsInterface) {
	msg, err := nats.Request("smartie.laptop."+device.NodeName+".charge", []byte(status), time.Second*5)

	if err == nil && string(msg.Data) == "ok" {
		device.IsCharging = status == "on"
	} else {
		log.Println(err)
	}
}

func (device *BatteryPoweredDevice) setBatteryMaintainLevel(level int, nats NatsInterface) {
	msg, err := nats.Request("smartie.laptop."+device.NodeName+".maintain", []byte(fmt.Sprint(level)), time.Second*5)

	if err == nil && string(msg.Data) == "ok" {
		device.MaintainLevel = level
	} else {
		log.Println(err)
	}
}
