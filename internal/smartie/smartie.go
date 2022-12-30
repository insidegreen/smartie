package smartie

import (
	"encoding/json"
	"errors"
	"log"
	"smarties/internal/promwatch"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type NatsInterface interface {
	Publish(subj string, data []byte) error
	Request(subj string, data []byte, timeout time.Duration) (*nats.Msg, error)
}

var natsConn *nats.Conn
var deviceMap map[string]*PlugDeviceInfo = make(map[string]*PlugDeviceInfo)
var promWatcher promwatch.PrometheusWatcher

func Operate() {
	nc, err := nats.Connect("192.168.86.33")
	promWatcher = promwatch.New("http://192.168.178.22:9090")

	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			maintainBatteryPoweredDevices()
			time.Sleep(time.Second * 10)
		}
	}()

	natsConn = nc

	apower := promauto.NewGauge(prometheus.GaugeOpts{
		Name:        "smartie_active_power_watts",
		ConstLabels: prometheus.Labels{"device": "pv"},
	})

	tasmotaPower := promauto.NewGauge(prometheus.GaugeOpts{
		Name:        "smartie_active_power_watts",
		ConstLabels: prometheus.Labels{"device": "tasmota"},
	})

	nc.Subscribe("shellies.pv.status.switch:0", func(m *nats.Msg) {
		var statusMsg ShellyStatus
		json.Unmarshal(m.Data, &statusMsg)

		apower.Set(statusMsg.Apower)
	})

	nc.Subscribe("shellies.plug.*.*.relay.>", func(m *nats.Msg) {
		var statusMsg ShellyStatus
		json.Unmarshal(m.Data, &statusMsg)

		deviceID := m.Subject[:strings.Index(m.Subject, ".relay")]
		currentPower, _ := strconv.ParseFloat(string(m.Data), 64)
		var deviceInfo *PlugDeviceInfo
		var deviceExists bool

		if deviceInfo, deviceExists = deviceMap[deviceID]; !deviceExists {
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
				pluggedDevice: BatteryPoweredDevice{
					NodeName:         getNodename(deviceID),
					BatteryLevel:     0,
					MaxMaintainLevel: 80,
					MinMaintainLevel: 20,
					IsAcPowered:      false,
					IsLaptop:         strings.HasPrefix(deviceID, "shellies.plug.laptop"),
				},
				actionCounter: 1,
			}
			deviceMap[deviceID] = deviceInfo
			if deviceInfo.pluggedDevice.IsLaptop {
				deviceInfo.priority = 1
			} else {
				deviceInfo.priority = 0.7
			}
		}

		if strings.HasSuffix(m.Subject, "power") {
			deviceInfo.promActivePowerGauge.Set(currentPower)
			deviceInfo.currentPower = currentPower
			deviceInfo.pluggedDevice.updatePowerConsumption(currentPower)
		} else if strings.HasSuffix(m.Subject, "relay.0") {
			deviceInfo.enabled = string(m.Data) == "on"
			if deviceInfo.enabled {
				deviceInfo.promEnabledGauge.Set(1)
			} else {
				deviceInfo.promEnabledGauge.Set(0)
			}
		}

	})

	nc.Subscribe("tele.*.SENSOR", func(m *nats.Msg) {
		var statusMsg TasmotaStatus
		json.Unmarshal(m.Data, &statusMsg)

		tasmotaPower.Set(statusMsg.SML.WattSumme)
		balance(statusMsg.SML.WattSumme)
	})

	nc.Flush()

	if err := nc.LastError(); err != nil {
		log.Fatal(err)
	}

}

func balance(overalWatt float64) error {

	if overalWatt > 0 {
		log.Printf("We're getting power from the grid(%f)\n", overalWatt)

		possibleDevice, err := getPowerOffCandidate(overalWatt)

		if err == nil {
			log.Printf("Turning OFF %s #%d", possibleDevice.mqqtSubject, possibleDevice.actionCounter)
			err = possibleDevice.setPlugStatus("off", natsConn)
			if err != nil {
				log.Print(err)
				return err
			}
			return nil
		} else {
			log.Print(err)
			return err
		}

	} else {
		log.Printf("We're spending power to the grid(%f)\n", overalWatt)

		possibleDevice, err := getPowerOnCandidate(overalWatt)

		if err == nil {
			log.Printf("Turning ON %s", possibleDevice.mqqtSubject)
			possibleDevice.setPlugStatus("on", natsConn)
			return nil
		} else {
			log.Print(err)
			return err
		}
	}
}

func getNodename(plugDeviceID string) string {
	nodenameSlice := strings.SplitAfter(plugDeviceID, ".")
	return nodenameSlice[len(nodenameSlice)-1]
}

func getPowerOffCandidate(overallWatt float64) (*PlugDeviceInfo, error) {
	var device *PlugDeviceInfo
	var priority float32 = -1

	for _, deviceInfo := range deviceMap {
		log.Printf("PlugID %s", deviceInfo.mqqtSubject)

		if deviceInfo.enabled && deviceInfo.pluggedDevice.BatteryLevel > deviceInfo.pluggedDevice.MaintainLevel {
			calcPrio := float32(deviceInfo.priority) / float32(deviceInfo.actionCounter)
			if deviceInfo.pluggedDevice.IsLaptop {
				calcPrio = calcPrio / (float32(deviceInfo.pluggedDevice.BatteryLevel) / 100)
			}
			if priority == -1 || calcPrio < float32(priority) {
				priority = calcPrio
				device = deviceInfo
			}
		}

	}

	if device == nil {
		return nil, errors.New("no candidate found")
	}

	return device, nil
}

func getPowerOnCandidate(overallWatt float64) (*PlugDeviceInfo, error) {
	var device *PlugDeviceInfo
	var priority float32 = 0.0

	for _, deviceInfo := range deviceMap {

		if !deviceInfo.enabled {
			calcPrio := float32(deviceInfo.priority) / float32(deviceInfo.actionCounter)
			if deviceInfo.pluggedDevice.IsLaptop {
				calcPrio = calcPrio / (float32(deviceInfo.pluggedDevice.BatteryLevel) / 100)
			}
			if calcPrio > float32(priority) {
				priority = calcPrio
				device = deviceInfo
			}
		}

	}

	if device == nil {
		return nil, errors.New("No candidate found!")
	}

	return device, nil
}
