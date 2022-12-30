package charger

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"smarties/internal/smartie"
	"strings"

	"github.com/nats-io/nats.go"
	dto "github.com/prometheus/client_model/go"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type StatusUpdater struct {
	natsConn   *nats.Conn
	DeviceInfo *smartie.BatteryPoweredDevice
}

func NewStatusUpdater(natsConn *nats.Conn) *StatusUpdater {
	hostname, err := os.Hostname()
	hostname, _, _ = strings.Cut(hostname, ".")

	if err != nil {
		log.Fatal(err)
	}

	return &StatusUpdater{
		natsConn: natsConn,
		DeviceInfo: &smartie.BatteryPoweredDevice{
			NodeName:         hostname,
			BatteryLevel:     -1,
			IsAcPowered:      false,
			IsLaptop:         true,
			IsCharging:       false,
			MaintainLevel:    80,
			MinMaintainLevel: 20,
			MaxMaintainLevel: 80,
		},
	}

}

func (su *StatusUpdater) UpdateStatus() {

	metrics, err := parseMetrics()

	smartie.Fatal(err)

	su.DeviceInfo.IsAcPowered = toBool(getValue(metrics["node_power_supply_power_source_state"], "state", "AC Power"))
	su.DeviceInfo.BatteryLevel = int(getValue(metrics["node_power_supply_current_capacity"]))
	su.DeviceInfo.IsCharging = toBool(getValue(metrics["node_power_supply_charging"]))

	payload, err := json.Marshal(su.DeviceInfo)

	smartie.Fatal(err)

	subject := fmt.Sprintf("smartie.laptop.%s.status", su.DeviceInfo.NodeName)

	su.natsConn.Publish(subject, payload)
}

func parseMetrics() (map[string]*dto.MetricFamily, error) {

	response, err := http.Get("http://localhost:9100/metrics")

	if err != nil {
		return nil, err
	}

	var parser expfmt.TextParser
	mf, err := parser.TextToMetricFamilies(response.Body)
	if err != nil {
		return nil, err
	}

	return mf, nil
}

func toBool(val float64) bool {
	return val == 1
}

func getValue(m *io_prometheus_client.MetricFamily, labelNameValue ...string) float64 {
	for _, v := range m.Metric {

		if len(labelNameValue) == 0 {
			return *v.Gauge.Value
		}

		for _, label := range v.Label {
			for iLNV := range labelNameValue {
				if iLNV%2 != 0 {
					continue
				}
				if *label.Name == labelNameValue[iLNV] && *label.Value == labelNameValue[iLNV+1] {
					return *v.Gauge.Value
				}
			}

		}
	}

	return -1
}
