package adapters

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/api/idtoken"

	"cloud.google.com/go/storage"
)

type GoogleCloud struct {
	Aud string
}

func NewGoogleCloud(aud string) *GoogleCloud {
	return &GoogleCloud{
		Aud: aud,
	}
}

func (gc *GoogleCloud) GetStorageClient() (interface{}, error) {
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

func (gc *GoogleCloud) GetIAPToken() (string, error) {
	ctx := context.Background()
	ts, err := idtoken.NewTokenSource(ctx, gc.Aud)
	if err != nil {
		log.Fatal(err)
	}
	token, err := ts.Token()
	return token.AccessToken, err
}
