package charger

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"smarties/internal/smartie"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

const chargeBatteryOn = "00"
const chargeBatteryOff = "02"

func Operate(deviceInfo *smartie.BatteryPoweredDevice, nc *nats.Conn) {

	hostname, err := os.Hostname()
	hostname, _, _ = strings.Cut(hostname, ".")

	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Hostname: %s", strings.ToLower(hostname))

	subject := fmt.Sprintf("smartie.laptop.%s.charge", hostname)
	nc.Subscribe(subject, func(msg *nats.Msg) {

		if deviceInfo.SmartChargeEnabled {
			switch string(msg.Data) {
			case "on":
				setBatteryStatus(deviceInfo, chargeBatteryOff)
			case "off":
				setBatteryStatus(deviceInfo, chargeBatteryOff)
			}
		}
	})

	subject = fmt.Sprintf("smartie.laptop.%s.maintain", hostname)
	nc.Subscribe(subject, func(msg *nats.Msg) {

		level := string(msg.Data)

		deviceInfo.MaintainLevel, err = strconv.Atoi(level)
		log.Printf("New maintain lvl is %d", deviceInfo.MaintainLevel)

		msg.Respond([]byte("ok"))
	})

	subject = fmt.Sprintf("smartie.laptop.%s.smart-charge", hostname)
	nc.Subscribe(subject, func(msg *nats.Msg) {
		switch string(msg.Data) {
		case "on":
			deviceInfo.SmartChargeEnabled = true
		case "off":
			deviceInfo.SmartChargeEnabled = false
		}
	})

	go chargeBattery(deviceInfo)

}

func chargeBattery(deviceInfo *smartie.BatteryPoweredDevice) error {

	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Printf("Current battery status is %d - charge lvl is %d ", deviceInfo.BatteryLevel, deviceInfo.MaintainLevel)
			if deviceInfo.BatteryLevel >= deviceInfo.MaintainLevel {
				setBatteryStatus(deviceInfo, chargeBatteryOff)
			} else if !deviceInfo.SmartChargeEnabled {
				setBatteryStatus(deviceInfo, chargeBatteryOn)
			}

		}
	}

}

func setBatteryStatus(deviceInfo *smartie.BatteryPoweredDevice, chargeFlag string) error {

	if deviceInfo.IsCharging && chargeFlag == chargeBatteryOn {
		return nil
	}
	if !deviceInfo.IsCharging && chargeFlag == chargeBatteryOff {
		return nil
	}

	if chargeFlag == chargeBatteryOn {
		log.Println("charging battery")
	} else {
		log.Println("disable battery charging")
	}

	for _, cmd := range []string{"CH0B", "CH0C"} {
		smc := exec.Command("/usr/local/bin/smc", "-k", cmd, "-w", chargeFlag)
		_, err := smc.CombinedOutput()

		if err != nil {
			log.Println(err)
			return err
		}

	}
	return nil
}
