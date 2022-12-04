package smartie

import (
	"log"
	"testing"
	"time"
)

var device1 *DeviceInfo = &DeviceInfo{
	mqqtSubject:   "device1",
	currentPower:  100,
	enabled:       false,
	priority:      1,
	actionCounter: 1,
	pluggedDevice: BatteryPoweredDevice{
		Nodename:     "device1laptop",
		BatteryPower: 21,
		AcPowered:    false,
		IsLaptop:     true,
	},
}

var device2 *DeviceInfo = &DeviceInfo{
	mqqtSubject:   "device2",
	currentPower:  100,
	enabled:       false,
	priority:      1,
	actionCounter: 1,
	pluggedDevice: BatteryPoweredDevice{
		Nodename:     "device2laptop",
		BatteryPower: 20,
		AcPowered:    false,
		IsLaptop:     true,
	},
}

func TestPowerOffNoEnabledPlugs(t *testing.T) {

	deviceMap["device1"] = device1
	deviceMap["device2"] = device2

	_, err := getPowerOffCandidate(200)

	if err == nil {
		t.Errorf("We should have an error, telling us that there is no candidate")
		log.Printf("error %s", err)
	}

}

func TestPowerOffOneEnabledPlug(t *testing.T) {
	deviceMap["device1"] = device1
	deviceMap["device2"] = device2

	device1.enabled = true
	device1.pluggedDevice.AcPowered = true
	candidate, _ := getPowerOffCandidate(200)

	if candidate != device1 {
		t.Errorf("expected device1")
	}
}

func TestPowerOffTwoEnabledPlugs(t *testing.T) {
	deviceMap["device1"] = device1
	deviceMap["device2"] = device2

	device1.enabled = true
	device1.pluggedDevice.AcPowered = true
	device2.enabled = true
	device2.pluggedDevice.AcPowered = true
	candidate, _ := getPowerOffCandidate(200)

	if candidate != device1 {
		t.Errorf("expected device1 ... got %v", candidate)
	}
}

func TestPowerOnNoEnabledPlugs(t *testing.T) {
	deviceMap["device1"] = device1
	deviceMap["device2"] = device2

	device1.enabled = false
	device1.pluggedDevice.AcPowered = false
	device2.enabled = false
	device2.pluggedDevice.AcPowered = false
	candidate, _ := getPowerOnCandidate(200)

	if candidate != device2 {
		t.Errorf("expected device2 ... got %v", candidate)
	}
}

func TestPowerOnOneEnabledPlugs(t *testing.T) {
	deviceMap["device1"] = device1
	deviceMap["device2"] = device2

	device1.enabled = false
	device1.pluggedDevice.AcPowered = false
	device2.enabled = true
	device2.pluggedDevice.AcPowered = true
	candidate, _ := getPowerOnCandidate(200)

	if candidate != device1 {
		t.Errorf("expected device1 ... got %v", candidate)
	}
}

type NatsMock struct {
	t      *testing.T
	device *DeviceInfo
}

func (nm *NatsMock) Publish(subj string, data []byte) error {
	nm.device.enabled = "on" == string(data)
	return nil
}

func TestPowerSwitch(t *testing.T) {
	deviceMap["device1"] = device1
	deviceMap["device2"] = device2

	device1.enabled = false
	device1.pluggedDevice.AcPowered = false

	setPlugStatus(device1, "on", &NatsMock{t: t, device: device1})
	actionTime := device1.actionTimestamp
	setPlugStatus(device1, "on", &NatsMock{t: t, device: device1})

	if actionTime != device1.actionTimestamp {
		t.Errorf("actionTimestamp has changed")
	}

	if device1.actionCounter != 2 {
		t.Errorf("Expected actionCounter == 2  ... got %d", device1.actionCounter)
	}

	setPlugStatus(device1, "off", &NatsMock{t: t, device: device1})
	if !device1.enabled {
		t.Errorf("Expected plug status still to be enabled")
	}

	device1.actionTimestamp = time.Now().Add(time.Minute * -9)
	setPlugStatus(device1, "off", &NatsMock{t: t, device: device1})
	if !device1.enabled {
		t.Errorf("Expected plug status still to be enabled")
	}

	device1.actionTimestamp = time.Now().Add(time.Minute * -11)
	setPlugStatus(device1, "off", &NatsMock{t: t, device: device1})
	if device1.enabled {
		t.Errorf("Expected plug status to be disabled")
	}
}
