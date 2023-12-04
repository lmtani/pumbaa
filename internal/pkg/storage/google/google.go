package google

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
)

func GetClient() (*storage.Client, error) {
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
