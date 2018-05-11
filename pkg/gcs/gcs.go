package gcs

import (
	"context"
	"io"
	"net/url"

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

/*
 * NewWriter creates a new writer on GCS for the given path.
 */
func NewWriter(client *storage.Client, path string) (io.WriteCloser, error) {
	bucket, path, err := splitPath(path)
	if err != nil {
		return nil, errors.Wrap(err, "split path")
	}
	ctx := context.Background()
	writer := client.Bucket(bucket).Object(path).NewWriter(ctx)
	return writer, nil
}

/*
 * NewReader creates a new reader on GCS for the given path.
 */
func NewReader(client *storage.Client, path string) (io.ReadCloser, error) {
	bucket, path, err := splitPath(path)
	if err != nil {
		return nil, errors.Wrap(err, "split path")
	}
	ctx := context.Background()
	reader, err := client.Bucket(bucket).Object(path).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	return reader, nil
}

/*
 * DeleteFile deletes a file from gcs
 */
func DeleteFile(client *storage.Client, path string) error {
	bucket, path, err := splitPath(path)
	if err != nil {
		return errors.Wrap(err, "split path")
	}
	ctx := context.Background()
	err = client.Bucket(bucket).Object(path).Delete(ctx)
	if err != nil {
		return errors.Wrap(err, "gcs")
	}
	return nil
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
