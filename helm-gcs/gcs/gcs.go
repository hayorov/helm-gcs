package gcs

import (
	"context"
	"errors"
	"io"
	"net/url"

	"cloud.google.com/go/storage"
)

func NewWriter(url string) (io.WriteCloser, error) {
	bucket, fullname, err := splitURL(url)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	b := client.Bucket(bucket)
	o := b.Object(fullname[1:])
	return o.NewWriter(ctx), nil
}

func NewReader(url string) (io.ReadCloser, error) {
	bucket, fullname, err := splitURL(url)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	b := client.Bucket(bucket)
	o := b.Object(fullname[1:])
	r, err := o.NewReader(ctx)
	return r, err
}

func splitURL(gcsurl string) (string, string, error) {
	u, err := url.Parse(gcsurl)
	if err != nil {
		return "", "", err
	}
	if u.Scheme != "gs" && u.Scheme != "gcs" {
		return "", "", errors.New(`incorrect url, should be "gs://bucket/path"`)
	}
	return u.Host, u.Path, nil
}

func DeleteFile(url string) error {
	bucket, fullname, err := splitURL(url)
	if err != nil {
		return err
	}
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	b := client.Bucket(bucket)
	o := b.Object(fullname[1:])
	return o.Delete(ctx)
}
