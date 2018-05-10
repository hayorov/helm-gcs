package gcs

import (
	"context"

	"cloud.google.com/go/storage"
	"github.com/pkg/errors"
	"google.golang.org/api/option"
)

func NewClient(serviceAccountPath string) (*storage.Client, error) {
	opts := []option.ClientOption{}
	if serviceAccountPath != "" {
		opts = append(opts, option.WithServiceAccountFile(serviceAccountPath))
	}
	client, err := storage.NewClient(context.Background(), opts...)
	if err != nil {
		return nil, errors.Wrap(err, "new client")
	}
	return client, err
}
