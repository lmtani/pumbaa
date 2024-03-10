package adapters

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
)

type GoogleStorage struct{}

func NewGoogleStorage() *GoogleStorage {
	return &GoogleStorage{}
}

func (gs *GoogleStorage) GetClient() (interface{}, error) {
	ctx := context.Background()

	// Attempt to create a client
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// If the client is created successfully, credentials are available
	fmt.Println("Credentials are available for Google Cloud Storage")
	return client, nil
}
