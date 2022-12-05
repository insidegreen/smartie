package promwatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type PrometheusWatcher struct {
	address string
	client  api.Client
}

func New(prometheusAddress string) PrometheusWatcher {

	pw := PrometheusWatcher{
		address: prometheusAddress,
	}
	client, err := api.NewClient(api.Config{
		Address: prometheusAddress,
	})

	if err != nil {
		log.Fatal("Could not initialize prom client!", err)
	}

	pw.client = client

	go func() { pw.operate() }()

	return pw
}

func (promWatcher *PrometheusWatcher) operate() {
	for {

		time.Sleep(time.Millisecond * 1000 * 5)
	}
}

func (promWatcher *PrometheusWatcher) PromQuery(query string) (float64, error) {

	v1api := v1.NewAPI(promWatcher.client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	timeRange := v1.Range{
		Start: time.Now().Add((-24 * time.Hour)),
		End:   time.Now(),
		Step:  time.Minute * 2,
	}

	result, warnings, err := v1api.QueryRange(ctx, query, timeRange, v1.WithTimeout(5*time.Second))
	if err != nil {
		fmt.Printf("Error querying Prometheus: %v\n", err)
		return -1, err
	}
	if len(warnings) > 0 {
		fmt.Printf("Warnings: %v\n", warnings)
	}

	switch r := result.(type) {
	case model.Vector:
		if len(r) > 0 {
			value, _ := strconv.ParseFloat(r[0].Value.String(), 64)

			log.Printf("Query result: %f", value)
			log.Printf("Metric result: %v", r[0].Metric)

			jsonMap := make(map[string](interface{}))

			byteJson, _ := r[0].MarshalJSON()

			log.Printf("Json: %s", string(byteJson))

			err := json.Unmarshal(byteJson, &jsonMap)

			if err != nil {
				log.Printf("ERROR: fail to unmarshla json, %s", err.Error())
			}

			log.Printf("Nodename %v", jsonMap["metric"])

			metric, _ := jsonMap["metric"].(map[string]interface{})
			log.Printf("Nodename %v", metric["nodename"])
		}
	case model.Matrix:
		if len(r) > 0 && len(r[len(r)-1].Values) > 0 {
			value, _ := strconv.ParseFloat(r[len(r)-1].Values[len(r[len(r)-1].Values)-1].Value.String(), 64)
			return value, nil
		}
	default:
		log.Printf("%v", r.Type())
		return -1, errors.New("Can't handle Prometheus result type " + r.Type().String())
	}

	return -1, errors.New("something strange happend")
}
