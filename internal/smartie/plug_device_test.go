package smartie

import (
	"testing"
	"time"
)

func TestPowerSwitch(t *testing.T) {
	plugDeviceMap["device1"] = device1
	plugDeviceMap["device2"] = device2

	device1.enabled = false
	device1.pluggedDevice.IsAcPowered = false

	device1.setPlugStatus("on", &NatsMock{t: t, device: device1})
	actionTime := device1.actionTimestamp
	device1.setPlugStatus("on", &NatsMock{t: t, device: device1})

	if actionTime != device1.actionTimestamp {
		t.Errorf("actionTimestamp has changed")
	}

	if device1.actionCounter != 2 {
		t.Errorf("Expected actionCounter == 2  ... got %d", device1.actionCounter)
	}

	device1.setPlugStatus("off", &NatsMock{t: t, device: device1})
	if !device1.enabled {
		t.Errorf("Expected plug status still to be enabled")
	}

	device1.actionTimestamp = time.Now().Add(time.Minute * -9)
	device1.setPlugStatus("off", &NatsMock{t: t, device: device1})
	if !device1.enabled {
		t.Errorf("Expected plug status still to be enabled")
	}

	device1.actionTimestamp = time.Now().Add(time.Minute * -11)
	device1.setPlugStatus("off", &NatsMock{t: t, device: device1})
	if device1.enabled {
		t.Errorf("Expected plug status to be disabled")
	}
}
