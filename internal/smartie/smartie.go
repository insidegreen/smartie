package smartie

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"smarties/internal/promwatch"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type ShellyStatus struct {
	ID      int     `json:"id"`
	Source  string  `json:"source"`
	Output  bool    `json:"output"`
	Apower  float64 `json:"apower"`
	Voltage float64 `json:"voltage"`
	Current float64 `json:"current"`
	Aenergy struct {
		Total    float64   `json:"total"`
		ByMinute []float64 `json:"by_minute"`
		MinuteTs int       `json:"minute_ts"`
	} `json:"aenergy"`
	Temperature struct {
		TC float64 `json:"tC"`
		TF float64 `json:"tF"`
	} `json:"temperature"`
}

type NatsInterface interface {
	Publish(subj string, data []byte) error
}

type TasmotaStatus struct {
	Time string `json:"Time"`
	SML  struct {
		VerbrauchT1      float64 `json:"Verbrauch_T1"`
		VerbrauchT2      float64 `json:"Verbrauch_T2"`
		VerbrauchSumme   float64 `json:"Verbrauch_Summe"`
		EinspeisungSumme float64 `json:"Einspeisung_Summe"`
		WattL1           float64 `json:"Watt_L1"`
		WattL2           float64 `json:"Watt_L2"`
		WattL3           float64 `json:"Watt_L3"`
		WattSumme        float64 `json:"Watt_Summe"`
		VoltL1           float64 `json:"Volt_L1"`
		VoltL2           float64 `json:"Volt_L2"`
		VoltL3           float64 `json:"Volt_L3"`
	} `json:"SML"`
}

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

type BatteryPoweredDevice struct {
	Nodename     string
	BatteryPower float64
	AcPowered    bool
	IsLaptop     bool
}

var natsConn *nats.Conn
var deviceMap map[string]*DeviceInfo = make(map[string]*DeviceInfo)
var promWatcher promwatch.PrometheusWatcher

func Operate() {
	nc, err := nats.Connect("192.168.86.33")
	promWatcher = promwatch.New("http://192.168.178.22:9090")

	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			pullBatteryPoweredDevices()
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
		var deviceInfo *DeviceInfo
		var deviceExists bool

		if deviceInfo, deviceExists = deviceMap[deviceID]; !deviceExists {
			deviceInfo = &DeviceInfo{
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
					Nodename:     getNodename(deviceID),
					BatteryPower: 0,
					AcPowered:    false,
					IsLaptop:     strings.HasPrefix(deviceID, "shellies.plug.laptop"),
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
			err = setPlugStatus(possibleDevice, "off", natsConn)
			if err != nil {
				return err
			}
			return nil
		} else {
			return err
		}

	} else {
		log.Printf("We're spending power to the grid(%f)\n", overalWatt)

		possibleDevice, err := getPowerOnCandidate(overalWatt)

		if err == nil {

			log.Printf("Turning ON %s", possibleDevice.mqqtSubject)
			setPlugStatus(possibleDevice, "on", natsConn)
			return nil
		} else {
			return err
		}
	}
}

func setPlugStatus(deviceInfo *DeviceInfo, status string, natsConn NatsInterface) error {

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
		natsConn.Publish(deviceInfo.mqqtSubject+".relay.0.command", []byte(status))
	} else {
		return errors.New("Unknow plug status " + status)
	}

	if status == "on" {
		deviceInfo.actionCounter++
		deviceInfo.actionTimestamp = time.Now()
	}

	return nil
}

func pullBatteryPoweredDevices() {
	for deviceId, deviceInfo := range deviceMap {
		log.Printf("PlugID %s", deviceId)

		query := fmt.Sprintf(`node_power_supply_current_capacity * on(instance) group_left(nodename) node_uname_info{nodename="%s"}`, getNodename(deviceId))
		queryResult, err := promWatcher.PromQuery(query)

		if err == nil {
			deviceInfo.pluggedDevice.BatteryPower = queryResult

			if queryResult <= 20 {
				setPlugStatus(deviceInfo, "on", natsConn)
			}
		}

		query = fmt.Sprintf(`node_power_supply_power_source_state{state=~"AC Power"} * on(instance) group_left(nodename) node_uname_info{nodename="%s"}`, getNodename(deviceId))
		queryResult, err = promWatcher.PromQuery(query)

		if err == nil {
			if queryResult == 1 {
				deviceInfo.pluggedDevice.AcPowered = true
			} else {
				deviceInfo.pluggedDevice.AcPowered = false
			}

		}

	}
}

func getNodename(plugDeviceID string) string {
	nodenameSlice := strings.SplitAfter(plugDeviceID, ".")
	return nodenameSlice[len(nodenameSlice)-1]
}

func getPowerOffCandidate(overallWatt float64) (*DeviceInfo, error) {
	var device *DeviceInfo
	var priority float32 = -1

	for _, deviceInfo := range deviceMap {
		log.Printf("PlugID %v", deviceInfo)

		if deviceInfo.enabled {
			calcPrio := float32(deviceInfo.priority) / float32(deviceInfo.actionCounter)
			if deviceInfo.pluggedDevice.IsLaptop {
				calcPrio = calcPrio / (float32(deviceInfo.pluggedDevice.BatteryPower) / 100)
			}
			if priority == -1 || calcPrio < float32(priority) {
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

func getPowerOnCandidate(overallWatt float64) (*DeviceInfo, error) {
	var device *DeviceInfo
	var priority float32 = 0.0

	for _, deviceInfo := range deviceMap {

		if !deviceInfo.enabled {
			calcPrio := float32(deviceInfo.priority) / float32(deviceInfo.actionCounter)
			if deviceInfo.pluggedDevice.IsLaptop {
				calcPrio = calcPrio / (float32(deviceInfo.pluggedDevice.BatteryPower) / 100)
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

/**





**/
