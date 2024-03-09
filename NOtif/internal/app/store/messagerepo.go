package store

import (
	"context"
	"notif/internal/app/model"
)

type MessageRepository struct {
	store *Store
}

func (r *MessageRepository) Create(model *model.Message) error {
	r.store.client.Database(r.store.config.DataBaseName).Collection(r.store.config.CollectionName).InsertOne(context.TODO(), model)
	return nil
}
