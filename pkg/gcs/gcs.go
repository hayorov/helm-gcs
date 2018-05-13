package gcs

import (
	"context"
	"net/url"

	"cloud.google.com/go/storage"
	"github.com/pkg/errors"
	"google.golang.org/api/option"
)

// NewClient creates a new gcs client.
// Use Application Default Credentials if serviceAccount is empty.
func NewClient(serviceAccountPath string) (*storage.Client, error) {
	opts := []option.ClientOption{}
	if serviceAccountPath != "" {
		opts = append(opts, option.WithCredentialsFile(serviceAccountPath))
	}
	client, err := storage.NewClient(context.Background(), opts...)
	if err != nil {
		return nil, errors.Wrap(err, "new client")
	}
	return client, err
}

// Object retourne a new object handle for the given path
func Object(client *storage.Client, path string) (*storage.ObjectHandle, error) {
	bucket, path, err := splitPath(path)
	if err != nil {
		return nil, errors.Wrap(err, "split path")
	}
	return client.Bucket(bucket).Object(path), nil
}

func splitPath(gcsurl string) (bucket string, path string, err error) {
	u, err := url.Parse(gcsurl)
	if err != nil {
		return
	}
	if u.Scheme != "gs" && u.Scheme != "gcs" {
		return "", "", errors.New(`incorrect url, should be "gs://bucket/path"`)
	}
	bucket = u.Host
	path = u.Path[1:]
	return
}
