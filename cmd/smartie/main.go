package main

import (
	"net/http"
	"smarties/internal/smartie"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	go smartie.Operate()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)

}
