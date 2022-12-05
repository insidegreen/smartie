package smartie

import (
	"fmt"
	"log"
	"time"
)

type BatteryPoweredDevice struct {
	Nodename           string
	BatteryLevel       int
	AcPowered          bool
	IsLaptop           bool
	MaintainLevel      int
	LowerMaintainLevel int
	UpperMaintainLevel int
	ChargePower        float64
	MaintainPower      float64
}

func maintainBatteryPoweredDevices() {
	for deviceId, deviceInfo := range deviceMap {
		log.Printf("PlugID %s", deviceId)

		batteryDevice := &deviceInfo.pluggedDevice

		query := fmt.Sprintf(`node_power_supply_power_source_state{state=~"AC Power"} * on(instance) group_left(nodename) node_uname_info{nodename="%s"}`, getNodename(deviceId))
		queryResult, err := promWatcher.PromQuery(query)

		if err == nil {
			if queryResult == 1 {
				batteryDevice.AcPowered = true
			} else {
				batteryDevice.AcPowered = false
			}
		}

		query = fmt.Sprintf(`node_power_supply_current_capacity * on(instance) group_left(nodename) node_uname_info{nodename="%s"}`, getNodename(deviceId))
		queryResult, err = promWatcher.PromQuery(query)

		if err == nil {
			batteryDevice.BatteryLevel = int(queryResult)

			if queryResult <= float64(batteryDevice.LowerMaintainLevel) && !deviceInfo.pluggedDevice.AcPowered {
				deviceInfo.pluggedDevice.setBatteryMaintainLevel(batteryDevice.LowerMaintainLevel, natsConn)
				deviceInfo.setPlugStatus("on", natsConn)
			}
		}
	}
}

func (device *BatteryPoweredDevice) setBatteryMaintainLevel(level int, nats NatsInterface) {
	msg, err := nats.Request("smartie.laptop."+device.Nodename+".charge", []byte(fmt.Sprint(level)), time.Second*5)

	if err == nil && string(msg.Data) == "ok" {
		device.MaintainLevel = level
	} else {
		log.Println(err)
	}
}

func (device *BatteryPoweredDevice) updatePowerConsumption(currentPower float64) {

	if device.AcPowered {
		if device.BatteryLevel < device.MaintainLevel && device.ChargePower < currentPower {
			device.ChargePower = currentPower
		}
		if device.BatteryLevel >= device.MaintainLevel && device.MaintainPower < currentPower {
			device.MaintainPower = currentPower
		}
	}

}
