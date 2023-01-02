package smartie

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

type PlugDeviceInfo struct {
	mqqtSubject          string
	promActivePowerGauge prometheus.Gauge
	promEnabledGauge     prometheus.Gauge
	currentPower         float64
	enabled              bool
	pluggedDevice        *BatteryPoweredDevice
	priority             float32
	actionCounter        int
	actionTimestamp      time.Time
}

var plugDeviceMap map[string]*PlugDeviceInfo = make(map[string]*PlugDeviceInfo)

func updatePlugDevice(m *nats.Msg) {
	var statusMsg ShellyStatus
	json.Unmarshal(m.Data, &statusMsg)

	deviceID := m.Subject[:strings.Index(m.Subject, ".relay")]
	currentPower, _ := strconv.ParseFloat(string(m.Data), 64)
	var deviceInfo *PlugDeviceInfo
	var deviceExists bool

	if deviceInfo, deviceExists = plugDeviceMap[deviceID]; !deviceExists {
		deviceInfo = &PlugDeviceInfo{
			mqqtSubject: deviceID,
			promActivePowerGauge: promauto.NewGauge(prometheus.GaugeOpts{
				Name:        "smartie_active_power_watts",
				ConstLabels: prometheus.Labels{"device": deviceID},
			}),
			promEnabledGauge: promauto.NewGauge(prometheus.GaugeOpts{
				Name:        "smartie_enabled_status",
				ConstLabels: prometheus.Labels{"device": deviceID},
			}),
			pluggedDevice: nil,
			actionCounter: 1,
		}
		plugDeviceMap[deviceID] = deviceInfo

	}

	if strings.HasSuffix(m.Subject, "power") {
		deviceInfo.promActivePowerGauge.Set(currentPower)

		powerDiff := math.Abs(deviceInfo.currentPower - currentPower)
		deviceInfo.currentPower = currentPower

		if powerDiff > 3 { // new device was plugged
			devicePlugedEvent <- deviceInfo
		}

	} else if strings.HasSuffix(m.Subject, "relay.0") {
		deviceInfo.enabled = string(m.Data) == "on"
		if deviceInfo.enabled {
			deviceInfo.promEnabledGauge.Set(1)
		} else {
			deviceInfo.promEnabledGauge.Set(0)
		}
	}

}

func (deviceInfo *PlugDeviceInfo) setPlugStatus(status string, nats NatsInterface) error {

	if (deviceInfo.enabled && status == "on") || (!deviceInfo.enabled && status == "off") {
		return nil
	}

	if deviceInfo.actionCounter > 1 && status == "off" {
		if deviceInfo.actionTimestamp.Add(time.Minute * 10).After(time.Now()) {
			logrus.Infof("Ignoring new plug status %s for device %s", status, deviceInfo.mqqtSubject)
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
