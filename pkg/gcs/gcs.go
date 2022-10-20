package gcs

import (
	"context"
	"net/url"
	"os"

	"cloud.google.com/go/storage"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
)

// NewClient creates a new gcs client.
// Use Application Default Credentials if serviceAccount is empty.
// Ignores ADC or serviceAccount when GOOGLE_OAUTH_ACCESS_TOKEN env variable is exported.
func NewClient(serviceAccountPath string) (*storage.Client, error) {
	opts := []option.ClientOption{}
	token := os.Getenv("GOOGLE_OAUTH_ACCESS_TOKEN")
	envCreds := os.Getenv("GOOGLE_CREDENTIALS") // used by terraform google provider
	ignoreEnvCreds := os.Getenv("HELM_GCS_IGNORE_TERRAFORM_CREDS")
	if token != "" {
		token := &oauth2.Token{AccessToken: token}
		opts = append(opts, option.WithTokenSource(oauth2.StaticTokenSource(token)))
	} else if envCreds != "" && ignoreEnvCreds != "true" {
		opts = append(opts, option.WithCredentialsJSON([]byte(envCreds)))
	} else if serviceAccountPath != "" {
		opts = append(opts, option.WithCredentialsFile(serviceAccountPath))
	}
	client, err := storage.NewClient(context.Background(), opts...)
	if err != nil {
		return nil, errors.Wrap(err, "new client")
	}
	return client, err
}

// Object returns a new object handle for the given path
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
