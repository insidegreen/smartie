package main

import (
	"net/http"
	"os"
	"smarties/internal/homeassistant"
	"smarties/internal/smartie"
	"smarties/internal/util"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

var natsConn *nats.Conn

func init() {

	nc, err := nats.Connect("nats")

	util.Fatal(err)

	natsConn = nc

	hostname, err := os.Hostname()
	hostname, _, _ = strings.Cut(hostname, ".")

	util.Fatal(err)

	hook := util.NewLogger(hostname, "system", nc)
	logrus.AddHook(hook)

	logrus.SetFormatter(&easy.Formatter{
		TimestampFormat: "2006-01-02 15:04:05",
		LogFormat:       "%time% [%lvl%] %node% - %msg%\n",
	})

	// logrus show line number
	logrus.SetReportCaller(true)
}

func main() {

	go smartie.Operate(natsConn)
	go homeassistant.Operate(natsConn)

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)

}
