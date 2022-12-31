package smartie

import (
	"encoding/json"
	"errors"
	"log"
	"math"
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
var devicePlugedEvent chan *PlugDeviceInfo
var laptopAcEvent chan *BatteryPoweredDevice

func Operate() {

	devicePlugedEvent = make(chan *PlugDeviceInfo)
	laptopAcEvent = make(chan *BatteryPoweredDevice)

	nc, err := nats.Connect("192.168.86.33")

	if err != nil {
		log.Fatal(err)
	}
	natsConn = nc

	apower := promauto.NewGauge(prometheus.GaugeOpts{
		Name:        "smartie_active_power_watts",
		ConstLabels: prometheus.Labels{"device": "pv"},
	})

	tasmotaPower := promauto.NewGauge(prometheus.GaugeOpts{
		Name:        "smartie_active_power_watts",
		ConstLabels: prometheus.Labels{"device": "tasmota"},
	})

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
		balance(statusMsg.SML.WattSumme)
	})

	nc.Flush()

	if err := nc.LastError(); err != nil {
		log.Fatal(err)
	}

	var lastPlugedEvent int64
	var lastLaptopAcEvent int64
	var lastPlug *PlugDeviceInfo
	var lastLaptop *BatteryPoweredDevice

	for {
		select {
		case plug := <-devicePlugedEvent:
			lastPlugedEvent = time.Now().UnixMilli()
			lastPlug = plug
			log.Println("Plug Event")
		case laptop := <-laptopAcEvent:
			lastLaptopAcEvent = time.Now().UnixMilli()
			lastLaptop = laptop
			log.Println("Laptop Event")
		}

		if math.Abs(float64(lastPlugedEvent-lastLaptopAcEvent)) <= 1000*20 {
			lastPlug.pluggedDevice = lastLaptop
			log.Printf("%s plugged into %s", lastLaptop.NodeName, lastPlug.mqqtSubject)

			lastPlugedEvent = -1
			lastLaptopAcEvent = -1
			lastPlug = nil
			lastLaptop = nil
		} else if lastPlug != nil {
			// lastPlug.pluggedDevice = nil
		}
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

func getPowerOffCandidate(overallWatt float64) (*PlugDeviceInfo, error) {
	var device *PlugDeviceInfo
	var priority float32 = -1

	for _, deviceInfo := range plugDeviceMap {
		log.Printf("PlugID %s", deviceInfo.mqqtSubject)

		if deviceInfo.pluggedDevice != nil && deviceInfo.enabled && deviceInfo.pluggedDevice.BatteryLevel > deviceInfo.pluggedDevice.MaintainLevel {
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

	for _, deviceInfo := range plugDeviceMap {

		if !deviceInfo.enabled {
			calcPrio := float32(deviceInfo.priority) / float32(deviceInfo.actionCounter)
			if deviceInfo.pluggedDevice != nil && deviceInfo.pluggedDevice.IsLaptop {
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
