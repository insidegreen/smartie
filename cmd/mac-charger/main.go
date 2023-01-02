package main

import (
	"context"
	"smarties/internal/charger"
	"smarties/internal/util"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

var statusUpdater *charger.StatusUpdater

func init() {

	nc, err := nats.Connect("192.168.86.33")
	if err != nil {
		logrus.Fatal(err)
	}

	statusUpdater = charger.NewStatusUpdater(nc)
	hook := util.NewLogger(statusUpdater.DeviceInfo.NodeName, "laptop", nc)
	logrus.AddHook(hook)

	logrus.SetFormatter(&easy.Formatter{
		TimestampFormat: "2006-01-02 15:04:05",
		LogFormat:       "%time% [%lvl%] %node% - %msg%\n",
	})

	// logrus show line number
	logrus.SetReportCaller(true)

	charger.Operate(statusUpdater.DeviceInfo, nc)
}

func main() {

	logrus.Println("mac charger is starting")
	context := context.Background()

	for {
		select {
		case <-context.Done():
			logrus.Info("mac charger is exiting")
		case <-time.Tick(time.Second * 3):
			statusUpdater.UpdateStatus()
		}
	}
}
