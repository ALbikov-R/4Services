package apiserver

import (
	"encoding/json"
	"fmt"
	"log"
	"notif/internal/app/model"
	"notif/internal/app/store"
	"os"
	"os/signal"
	"sync"

	"github.com/IBM/sarama"
)

type Message struct {
	Value string
}

type Receiver struct {
	Consumer          sarama.Consumer
	PartitionConsumer sarama.PartitionConsumer
	Topic             string
	ShutdownSignal    chan os.Signal
	WaitGroup         sync.WaitGroup
	store             *store.Store
}

// NewReceiver создает новый экземпляр Receiver
func NewReceiver(brokerList []string, topic string) (*Receiver, error) {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	consumer, err := sarama.NewConsumer(brokerList, config)
	if err != nil {
		return nil, err
	}
	partitionConsumer, err := consumer.ConsumePartition(topic, 0, sarama.OffsetNewest)
	if err != nil {
		return nil, err
	}

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, os.Interrupt)

	return &Receiver{
		Consumer:          consumer,
		PartitionConsumer: partitionConsumer,
		Topic:             topic,
		ShutdownSignal:    shutdownSignal,
	}, nil
}

// HandleMessages обрабатывает входящие сообщения
func (r *Receiver) HandleMessages() {
	defer r.WaitGroup.Done()
	for {
		select {
		case msg := <-r.PartitionConsumer.Messages():
			fmt.Println(string(msg.Value))
			var message model.Message
			err := json.Unmarshal(msg.Value, &message)
			if err != nil {
				log.Fatal("Ошибка парсинга JSON файла из Kafka")
			}
			r.processMessage(message)
		case err := <-r.PartitionConsumer.Errors():
			log.Printf("Error: %v", err)
		case <-r.ShutdownSignal:
			r.PartitionConsumer.Close()
			return
		}
	}
}
func (r *Receiver) processMessage(message model.Message) {
	log.Printf("Received message: %s", message)
	r.store.Message().Create(&message)
}

func (r *Receiver) Start() {
	r.store = store.New()
	if r.store.Open() != nil {
		log.Fatal("Ошибка с открытием БД")
	}
	defer r.store.Close()
	r.WaitGroup.Add(1)
	go r.HandleMessages()
	r.WaitGroup.Wait()
}
