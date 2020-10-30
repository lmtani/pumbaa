package input

import (
	"context"
	"io"
	"os"

	"cloud.google.com/go/storage"
)

func ReadFile(bucketName, objName string) error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	bkt := client.Bucket(bucketName)
	obj := bkt.Object(objName)
	r, err := obj.NewReader(ctx)
	if err != nil {
		return err
	}
	defer r.Close()
	if _, err := io.Copy(os.Stdout, r); err != nil {
		return err
	}
	return nil
}
