package main

import (
	"log"
	"notif/internal/app/apiserver"
	"os"
)

func main() {
	brokerList := []string{os.Getenv("KAFKA_PORT")}
	topic := os.Getenv("TOPIC")

	receiver, err := apiserver.NewReceiver(brokerList, topic)
	if err != nil {
		log.Fatalf("Failed to initialize receiver: %v", err)
	}
	log.Println("Connection is available ")
	receiver.Start()
}
