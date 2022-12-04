package charger

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/nats-io/nats.go"
)

var batteryCharger *exec.Cmd

func ListenOnNats() {

	nc, err := nats.Connect("192.168.86.33")
	if err != nil {
		log.Fatal(err)
	}

	hostname, err := os.Hostname()
	hostname, _, _ = strings.Cut(hostname, ".")

	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Hostname: %s", strings.ToLower(hostname))

	subject := fmt.Sprintf("smartie.laptop.%s.charge", hostname)

	nc.Subscribe(subject, func(msg *nats.Msg) {

		level := string(msg.Data)
		go chargeBattery(level)

		msg.Respond([]byte("ok"))
	})

}

func chargeBattery(level string) error {

	if batteryCharger != nil {
		err := batteryCharger.Process.Kill()
		if err != nil {
			log.Println(err)
		}
	}

	batteryCharger = exec.Command("/usr/local/bin/battery", "charge", level)
	stdout, err := batteryCharger.StdoutPipe()
	batteryCharger.Stderr = batteryCharger.Stdout
	if err != nil {
		log.Println(err)
		return err
	}
	if err = batteryCharger.Start(); err != nil {
		log.Println(err)
		return err
	}

	defer log.Println("go routine ended. battery process finished")
	for {
		tmp := make([]byte, 1024)
		len, err := stdout.Read(tmp)
		log.Print(string(tmp[0:len]))
		if err != nil {
			return err
		}
	}
}
