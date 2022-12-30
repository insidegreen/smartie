package smartie

import (
	"fmt"
	"log"
	"time"
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

func maintainBatteryPoweredDevices() {
	for deviceId, deviceInfo := range deviceMap {
		log.Printf("PlugID %s", deviceId)

		batteryDevice := &deviceInfo.pluggedDevice

		query := fmt.Sprintf(`node_power_supply_power_source_state{state=~"AC Power"} * on(instance) group_left(nodename) node_uname_info{nodename="%s"}`, getNodename(deviceId))
		queryResult, err := promWatcher.PromQuery(query)

		if err == nil {
			if queryResult == 1 {
				batteryDevice.IsAcPowered = true
			} else {
				batteryDevice.IsAcPowered = false
			}
		}

		query = fmt.Sprintf(`node_power_supply_current_capacity * on(instance) group_left(nodename) node_uname_info{nodename="%s"}`, getNodename(deviceId))
		queryResult, err = promWatcher.PromQuery(query)

		if err == nil {
			batteryDevice.BatteryLevel = int(queryResult)

			if queryResult <= float64(batteryDevice.MinMaintainLevel) {
				deviceInfo.pluggedDevice.setBatteryMaintainLevel(batteryDevice.MinMaintainLevel, natsConn)
				deviceInfo.pluggedDevice.setBatteryChargeStatus("on", natsConn)
				if !deviceInfo.pluggedDevice.IsAcPowered {
					deviceInfo.setPlugStatus("on", natsConn)
				}
			}
		}
	}
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

func (device *BatteryPoweredDevice) updatePowerConsumption(currentPower float64) {

	// if device.IsAcPowered {
	// 	if device.BatteryLevel < device.MaintainLevel && device.ChargePower < currentPower {
	// 		device.ChargePower = currentPower
	// 	}
	// 	if device.BatteryLevel >= device.MaintainLevel && device.MaintainPower < currentPower {
	// 		device.MaintainPower = currentPower
	// 	}
	// }

}
