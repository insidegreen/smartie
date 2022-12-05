package smartie

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type DeviceInfo struct {
	mqqtSubject          string
	promActivePowerGauge prometheus.Gauge
	promEnabledGauge     prometheus.Gauge
	currentPower         float64
	enabled              bool
	pluggedDevice        BatteryPoweredDevice
	priority             float32
	actionCounter        int
	actionTimestamp      time.Time
}

func (deviceInfo *DeviceInfo) setPlugStatus(status string, nats NatsInterface) error {

	if (deviceInfo.enabled && status == "on") || (!deviceInfo.enabled && status == "off") {
		return nil
	}

	if deviceInfo.actionCounter > 1 && status == "off" {
		if deviceInfo.actionTimestamp.Add(time.Minute * 10).After(time.Now()) {
			log.Printf("Ignoring new plug status %s for device %s", status, deviceInfo.mqqtSubject)
			return fmt.Errorf("ignored! turning off device %s is blocked until %s", deviceInfo.mqqtSubject,
				deviceInfo.actionTimestamp.Add(time.Minute*10).Format("15:04:05"))
		}
	}

	if status == "on" || status == "off" {
		nats.Publish(deviceInfo.mqqtSubject+".relay.0.command", []byte(status))
	} else {
		return errors.New("Unknow plug status " + status)
	}

	if status == "on" {
		deviceInfo.actionCounter++
		deviceInfo.actionTimestamp = time.Now()
	}

	return nil
}
