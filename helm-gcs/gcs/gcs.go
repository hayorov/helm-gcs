package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"

	"cloud.google.com/go/storage"
)

func NewWriter(url string) (io.WriteCloser, error) {
	makeError := makeErrorFunc("gcs.NewWriter")
	bucket, fullname, err := splitURL(url)
	if err != nil {
		return nil, makeError(err)
	}
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, makeError(err)
	}
	debug("create new writer for file '%s' in bucket '%s' (url: %s)", fullname[1:], bucket, url)
	b := client.Bucket(bucket)
	o := b.Object(fullname[1:])
	return o.NewWriter(ctx), nil
}

func NewReader(url string) (io.ReadCloser, error) {
	makeError := makeErrorFunc("gcs.NewReader")
	bucket, fullname, err := splitURL(url)
	if err != nil {
		return nil, makeError(err)
	}
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, makeError(err)
	}
	debug("create new reader for file '%s' in bucket '%s' (url: %s)", fullname[1:], bucket, url)
	b := client.Bucket(bucket)
	o := b.Object(fullname[1:])
	r, err := o.NewReader(ctx)
	if err != nil {
		return nil, makeError(err)
	}
	return r, nil
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
	makeError := makeErrorFunc("gcs.DeleteFile")
	bucket, fullname, err := splitURL(url)
	if err != nil {
		return makeError(err)
	}
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return makeError(err)
	}
	debug("delete file '%s' in bucket '%s' (url: %s)", fullname[1:], bucket, url)
	b := client.Bucket(bucket)
	o := b.Object(fullname[1:])
	err = o.Delete(ctx)
	if err != nil {
		return makeError(err)
	}
	return nil
}

var Debug bool

func debug(str string, args ...interface{}) {
	str = "gcs: " + str
	if Debug {
		if len(args) == 0 {
			fmt.Println(str)
		} else {
			fmt.Printf(str+"\n", args...)
		}
	}
}

func makeErrorFunc(prefix string) func(error) error {
	return func(err error) error {
		return fmt.Errorf("%s: %s", prefix, err.Error())
	}
}
