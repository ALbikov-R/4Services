package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Store struct {
	client            *mongo.Client
	Config            *Config
	messageRepository *MessageRepository
}

func New() *Store {
	return &Store{
		Config: NewConfig(),
	}
}

func (s *Store) Open() error {
	clientOptions := options.Client().ApplyURI(s.Config.MongoURI)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var err error
	s.client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		return err
	}
	err = s.client.Ping(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) Close() {
	s.client.Disconnect(context.Background())
}
func (s *Store) Message() *MessageRepository {
	if s.messageRepository != nil {
		return s.messageRepository
	}
	s.messageRepository = &MessageRepository{
		store: s,
	}
	return s.messageRepository
}
