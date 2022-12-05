package smartie

import (
	"log"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

type NatsMock struct {
	t      *testing.T
	device *DeviceInfo
}

func (nm *NatsMock) Publish(subj string, data []byte) error {
	nm.device.enabled = "on" == string(data)
	return nil
}

func (nm *NatsMock) Request(subj string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	return &nats.Msg{
		Data: []byte("ok"),
	}, nil
}

var device1 *DeviceInfo = &DeviceInfo{
	mqqtSubject:   "device1",
	currentPower:  100,
	enabled:       false,
	priority:      1,
	actionCounter: 1,
	pluggedDevice: BatteryPoweredDevice{
		Nodename:     "device1laptop",
		BatteryLevel: 21,
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
		BatteryLevel: 20,
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
