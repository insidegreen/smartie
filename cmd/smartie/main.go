package main

import (
	"net/http"
	"smarties/internal/smartie"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	smartie.Operate()

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)

}
