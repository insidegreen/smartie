package charger

import (
	"fmt"
	"os/exec"
	"smarties/internal/smartie"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

const chargeBatteryOn = "00"
const chargeBatteryOff = "02"

func Operate(deviceInfo *smartie.BatteryPoweredDevice, nc *nats.Conn) {

	subject := fmt.Sprintf("smartie.laptop.%s.charge", deviceInfo.NodeName)
	nc.Subscribe(subject, func(msg *nats.Msg) {
		logrus.Infof("Got charge command: %s", string(msg.Data))

		response := "ign"

		if deviceInfo.SmartChargeEnabled {
			switch string(msg.Data) {
			case "on":
				response, _ = setBatteryStatus(deviceInfo, chargeBatteryOn)
			case "off":
				response, _ = setBatteryStatus(deviceInfo, chargeBatteryOff)
			}
		}

		error := msg.Respond([]byte(response))

		if error != nil {
			logrus.Errorf("Could not send nats reply: %s", error)
		}
	})

	subject = fmt.Sprintf("smartie.laptop.%s.maintain", deviceInfo.NodeName)
	nc.Subscribe(subject, func(msg *nats.Msg) {

		level, err := strconv.Atoi(string(msg.Data))

		if err == nil {
			deviceInfo.MaintainLevel = level
			logrus.Infof("New maintain lvl is %d", deviceInfo.MaintainLevel)

			msg.Respond([]byte("ok"))

		} else {
			msg.Respond([]byte("nok"))
		}
	})

	subject = fmt.Sprintf("smartie.laptop.%s.smart-charge", deviceInfo.NodeName)
	nc.Subscribe(subject, func(msg *nats.Msg) {
		switch strings.ToLower(string(msg.Data)) {
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
			logrus.Infof("Current battery status is %d - charge lvl is %d ", deviceInfo.BatteryLevel, deviceInfo.MaintainLevel)
			if deviceInfo.BatteryLevel >= deviceInfo.MaintainLevel {
				setBatteryStatus(deviceInfo, chargeBatteryOff)
			} else if !deviceInfo.SmartChargeEnabled {
				setBatteryStatus(deviceInfo, chargeBatteryOn)
			}

		}
	}

}

func setBatteryStatus(deviceInfo *smartie.BatteryPoweredDevice, chargeFlag string) (string, error) {

	if deviceInfo.IsCharging && chargeFlag == chargeBatteryOn {
		return "ign", nil
	}
	if !deviceInfo.IsCharging && chargeFlag == chargeBatteryOff {
		return "ign", nil
	}

	battStat, err := getBatteryStatus()

	if err != nil {
		logrus.Errorf("could not get battery status via smc - %s", err.Error())
		return "nok", err
	}

	if chargeFlag == chargeBatteryOn {
		if battStat == 0 {
			logrus.Info("battery is already charging")
			return "ign", nil
		}
		logrus.Info("charging battery")
	} else {
		if battStat == 3 {
			logrus.Info("battery charging is already disabled")
			return "ign", nil
		}
		logrus.Info("disable battery charging")
	}

	for _, cmd := range []string{"CH0B", "CH0C"} {
		smc := exec.Command("/usr/local/bin/smc", "-k", cmd, "-w", chargeFlag)
		_, err := smc.CombinedOutput()

		if err != nil {
			logrus.Errorf("smc %s %s returned an error %s", cmd, chargeFlag, err)
			return "nok", err
		}

	}
	return "ok", nil
}

func getBatteryStatus() (int, error) {

	smc := exec.Command("/usr/local/bin/smc", "-k", "CH0B", "-r")
	out, err := smc.CombinedOutput()

	if err != nil {
		logrus.Errorf("smc CH0B returned an error %s", err)
	} else {
		logrus.Info(string(out))
	}

	result := string(out)
	index := strings.Index(result, "(bytes ")

	if index+9 > len(result) {
		return -1, fmt.Errorf("got an invalid result from smc: %s", result)
	}

	return strconv.Atoi(result[index+7 : index+9])
}
