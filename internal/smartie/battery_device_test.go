package smartie

import (
	"testing"
)

func TestUpdatePowerConsumption(t *testing.T) {
	device := &DeviceInfo{
		mqqtSubject:   "device1",
		currentPower:  100,
		enabled:       true,
		priority:      1,
		actionCounter: 1,
		pluggedDevice: BatteryPoweredDevice{
			Nodename:      "device1laptop",
			BatteryLevel:  21,
			AcPowered:     true,
			IsLaptop:      true,
			MaintainLevel: 80,
			ChargePower:   0,
			MaintainPower: 0,
		},
	}

	pow := []float64{95, 100, 99, 110}
	exp := []float64{95, 100, 100, 110}
	level := []int{25, 85}
	var attr *float64

	for _, bLev := range level {
		device.pluggedDevice.BatteryLevel = bLev

		if bLev < device.pluggedDevice.MaintainLevel {
			attr = &device.pluggedDevice.ChargePower
		} else {
			attr = &device.pluggedDevice.MaintainPower
		}

		for c, v := range pow {
			device.pluggedDevice.updatePowerConsumption(v)
			if *attr != exp[c] {
				t.Errorf("Expected %f ... got %f for battery level %d", exp[c], *attr, bLev)
			}
		}
	}
}
