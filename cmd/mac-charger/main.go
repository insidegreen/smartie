package main

import (
	"context"
	"log"
	"smarties/internal/charger"
	"time"

	"github.com/nats-io/nats.go"
)

var statusUpdater *charger.StatusUpdater

func init() {

	nc, err := nats.Connect("192.168.86.33")
	if err != nil {
		log.Fatal(err)
	}

	statusUpdater = charger.NewStatusUpdater(nc)

	charger.Operate(statusUpdater.DeviceInfo, nc)
}

func main() {

	// logPath := "/Users/tgr/Library/Logs/Homebrew/smartie"

	// f, err := os.OpenFile(logPath+"/smartie.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	// if err != nil {
	// 	panic(err)
	// }
	// defer f.Close()

	// log.SetOutput(f)

	log.Println("This is a log from GOLANG")

	context := context.Background()

	for {
		select {
		case <-context.Done():
			log.Println("Exit")
		case <-time.Tick(time.Second * 3):
			statusUpdater.UpdateStatus()
		}
	}
}
