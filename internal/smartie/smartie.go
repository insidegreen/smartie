package smartie

import (
	"encoding/json"
	"errors"
	"math"
	"smarties/internal/util"
	"sort"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

type NatsInterface interface {
	Publish(subj string, data []byte) error
	Request(subj string, data []byte, timeout time.Duration) (*nats.Msg, error)
}

var natsConn *nats.Conn
var devicePlugedEvent chan *PlugDeviceInfo
var laptopAcEvent chan *BatteryPoweredDevice
var powerEvent chan float64

func Operate(nc *nats.Conn) {

	devicePlugedEvent = make(chan *PlugDeviceInfo)
	laptopAcEvent = make(chan *BatteryPoweredDevice)
	powerEvent = make(chan float64)

	natsConn = nc

	apower := promauto.NewGauge(prometheus.GaugeOpts{
		Name:        "smartie_active_power_watts",
		ConstLabels: prometheus.Labels{"device": "pv"},
	})

	tasmotaPower := promauto.NewGauge(prometheus.GaugeOpts{
		Name:        "smartie_active_power_watts",
		ConstLabels: prometheus.Labels{"device": "tasmota"},
	})

	nc.Subscribe("smartie.laptop.*.plug", setPlugStatus)
	nc.Subscribe("smartie.laptop.*.status", updateBatteryPoweredDevice)
	nc.Subscribe("shellies.plug.*.*.relay.>", updatePlugDevice)

	nc.Subscribe("shellies.pv.status.switch:0", func(m *nats.Msg) {
		var statusMsg ShellyStatus
		json.Unmarshal(m.Data, &statusMsg)

		apower.Set(statusMsg.Apower)
	})

	nc.Subscribe("tele.*.SENSOR", func(m *nats.Msg) {
		var statusMsg TasmotaStatus
		json.Unmarshal(m.Data, &statusMsg)

		tasmotaPower.Set(statusMsg.SML.WattSumme)
		powerEvent <- statusMsg.SML.WattSumme
	})

	nc.Flush()

	if err := nc.LastError(); err != nil {
		util.Fatal(err)
	}

	go detectPlugLaptopRelation()
	go balance()
}

func detectPlugLaptopRelation() {
	var lastPlugedEvent int64
	var lastLaptopAcEvent int64
	var lastPlug *PlugDeviceInfo
	var lastLaptop *BatteryPoweredDevice

	for {
		select {
		case plug := <-devicePlugedEvent:
			lastPlugedEvent = time.Now().UnixMilli()
			lastPlug = plug
		case laptop := <-laptopAcEvent:
			lastLaptopAcEvent = time.Now().UnixMilli()
			lastLaptop = laptop
		}

		eventDiff := math.Abs(float64(lastPlugedEvent - lastLaptopAcEvent))

		if eventDiff <= 1000*10 {

			for _, pdi := range plugDeviceMap {
				if pdi.pluggedDevice != nil && pdi.pluggedDevice.NodeName == lastLaptop.NodeName {
					pdi.pluggedDevice = nil
					pdi.priority = 0.7
				}
			}

			lastPlug.pluggedDevice = lastLaptop
			lastPlug.priority = 0.7
			logrus.Infof("%s plugged into %s", lastLaptop.NodeName, lastPlug.mqqtSubject)

			lastPlugedEvent = -1
			lastLaptopAcEvent = -1
			lastPlug = nil
			lastLaptop = nil
		}
	}

}

func balance() {

mainLoop:
	for overalWatt := range powerEvent {

		if overalWatt > 0 {
			logrus.Infof("We're getting power from the grid(%f)\n", overalWatt)

			drainableDevices := getDrainableCandidate()

			for _, dd := range drainableDevices {
				if dd.setBatteryChargeStatus("off", natsConn) == nil {
					continue mainLoop
				}
			}

			possibleDevice, err := getPowerOffCandidate(overalWatt)

			if err == nil {
				logrus.Infof("Turning OFF %s #%d", possibleDevice.mqqtSubject, possibleDevice.actionCounter)
				err = possibleDevice.setPlugStatus("off", natsConn)
				if err != nil {
					logrus.Error(err)
				}
			} else {
				logrus.Error(err)
			}

		} else {
			logrus.Infof("We're spending power to the grid(%f)\n", overalWatt)

			chargeableDevices := getChargingCandidate()

			for _, dd := range chargeableDevices {
				if dd.setBatteryChargeStatus("on", natsConn) == nil {
					continue mainLoop
				}
			}

			possibleDevice, err := getPowerOnCandidate(overalWatt)

			if err == nil {
				logrus.Infof("Turning ON %s", possibleDevice.mqqtSubject)
				possibleDevice.setPlugStatus("on", natsConn)
			} else {
				logrus.Error(err)
			}
		}
	}

}

func getDrainableCandidate() []*BatteryPoweredDevice {

	var candidates []*BatteryPoweredDevice

	for _, can := range battDeviceMap {
		if !can.SmartChargeEnabled || can.BatteryLevel < can.MaintainLevel {
			continue
		}
		if can.IsAcPowered && can.IsCharging {
			candidates = append(candidates, can)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].BatteryLevel > candidates[j].BatteryLevel
	})
	return candidates
}

func getChargingCandidate() []*BatteryPoweredDevice {
	var candidates []*BatteryPoweredDevice

	for _, can := range battDeviceMap {
		if !can.SmartChargeEnabled {
			continue
		}
		if can.IsAcPowered && !can.IsCharging {
			candidates = append(candidates, can)
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].BatteryLevel < candidates[j].BatteryLevel
	})
	return candidates
}

func getPowerOffCandidate(overallWatt float64) (*PlugDeviceInfo, error) {
	var device *PlugDeviceInfo
	var priority float32 = -1

	for _, deviceInfo := range plugDeviceMap {
		pd := deviceInfo.pluggedDevice

		if pd == nil || !pd.SmartChargeEnabled {
			continue
		}

		// if the connected device has less battery than MaintainLevel, we need power on the plug
		if deviceInfo.enabled && pd.BatteryLevel > pd.MaintainLevel {

			calcPrio := float32(deviceInfo.priority) / float32(deviceInfo.actionCounter)
			if pd.IsLaptop {
				calcPrio = calcPrio / (float32(pd.BatteryLevel) / 100)
			}
			if priority == -1 || calcPrio < float32(priority) {
				priority = calcPrio
				device = deviceInfo
			}
		}

	}

	if device == nil {
		return nil, errors.New("no power-off candidate found")
	}

	return device, nil
}

func getPowerOnCandidate(overallWatt float64) (*PlugDeviceInfo, error) {
	var device *PlugDeviceInfo
	var priority float32 = 0.0

	for _, deviceInfo := range plugDeviceMap {

		if !deviceInfo.enabled && deviceInfo.pluggedDevice != nil {

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
		return nil, errors.New("no power-on candidate found")
	}

	return device, nil
}
