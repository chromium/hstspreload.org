package database

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
)

func ObjectWriter(name string) (*storage.Writer, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), scanAllTimeout)

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, cancel, fmt.Errorf("failed to create storage client: %v", err)
	}

	obj := client.Bucket(scanBucketName).Object(name)
	return obj.NewWriter(ctx), cancel, nil
}
