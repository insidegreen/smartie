package main

import (
	"context"
	"log"
	"smarties/internal/charger"
	"time"
)

func init() {
	charger.ListenOnNats()

}

func main() {

	context := context.Background()

	for {
		select {
		case <-context.Done():
			log.Println("Exit")
		case <-time.Tick(time.Second * 10):
		}
	}
}
