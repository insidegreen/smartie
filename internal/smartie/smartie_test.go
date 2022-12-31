package smartie

import (
	"log"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

type NatsMock struct {
	t      *testing.T
	device *PlugDeviceInfo
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

var device1 *PlugDeviceInfo = &PlugDeviceInfo{
	mqqtSubject:   "device1",
	currentPower:  100,
	enabled:       false,
	priority:      1,
	actionCounter: 1,
	pluggedDevice: &BatteryPoweredDevice{
		NodeName:     "device1laptop",
		BatteryLevel: 21,
		IsAcPowered:  false,
		IsLaptop:     true,
	},
}

var device2 *PlugDeviceInfo = &PlugDeviceInfo{
	mqqtSubject:   "device2",
	currentPower:  100,
	enabled:       false,
	priority:      1,
	actionCounter: 1,
	pluggedDevice: &BatteryPoweredDevice{
		NodeName:     "device2laptop",
		BatteryLevel: 20,
		IsAcPowered:  false,
		IsLaptop:     true,
	},
}

func TestPowerOffNoEnabledPlugs(t *testing.T) {

	plugDeviceMap["device1"] = device1
	plugDeviceMap["device2"] = device2

	_, err := getPowerOffCandidate(200)

	if err == nil {
		t.Errorf("We should have an error, telling us that there is no candidate")
		log.Printf("error %s", err)
	}

}

func TestPowerOffOneEnabledPlug(t *testing.T) {
	plugDeviceMap["device1"] = device1
	plugDeviceMap["device2"] = device2

	device1.enabled = true
	device1.pluggedDevice.IsAcPowered = true
	candidate, _ := getPowerOffCandidate(200)

	if candidate != device1 {
		t.Errorf("expected device1")
	}
}

func TestPowerOffTwoEnabledPlugs(t *testing.T) {
	plugDeviceMap["device1"] = device1
	plugDeviceMap["device2"] = device2

	device1.enabled = true
	device1.pluggedDevice.IsAcPowered = true
	device2.enabled = true
	device2.pluggedDevice.IsAcPowered = true
	candidate, _ := getPowerOffCandidate(200)

	if candidate != device1 {
		t.Errorf("expected device1 ... got %v", candidate)
	}
}

func TestPowerOnNoEnabledPlugs(t *testing.T) {
	plugDeviceMap["device1"] = device1
	plugDeviceMap["device2"] = device2

	device1.enabled = false
	device1.pluggedDevice.IsAcPowered = false
	device2.enabled = false
	device2.pluggedDevice.IsAcPowered = false
	candidate, _ := getPowerOnCandidate(200)

	if candidate != device2 {
		t.Errorf("expected device2 ... got %v", candidate)
	}
}

func TestPowerOnOneEnabledPlugs(t *testing.T) {
	plugDeviceMap["device1"] = device1
	plugDeviceMap["device2"] = device2

	device1.enabled = false
	device1.pluggedDevice.IsAcPowered = false
	device2.enabled = true
	device2.pluggedDevice.IsAcPowered = true
	candidate, _ := getPowerOnCandidate(200)

	if candidate != device1 {
		t.Errorf("expected device1 ... got %v", candidate)
	}
}
