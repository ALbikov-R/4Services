package main

import (
	"log"
	"notif/internal/app/apiserver"
)

func main() {
	brokerList := []string{"localhost:9092"}
	topic := "Order_test"

	receiver, err := apiserver.NewReceiver(brokerList, topic)
	if err != nil {
		log.Fatalf("Failed to initialize receiver: %v", err)
	}
	receiver.Start()
}
