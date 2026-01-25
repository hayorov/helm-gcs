// Copyright © 2024 helm-gcs contributors
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// helm-gcs-getter is a Helm 4 getter plugin for Google Cloud Storage.
// It handles gs:// protocol URLs and downloads objects to stdout.
//
// Helm 4 invokes getter plugins with:
//
//	<command> <certFile> <keyFile> <caFile> <href>
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hayorov/helm-gcs/pkg/gcs"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Printf("helm-gcs-getter %s (commit: %s, date: %s)\n", version, commit, date)
		os.Exit(0)
	}

	if len(os.Args) < 5 {
		fmt.Fprintf(os.Stderr, "helm-gcs-getter - Helm 4 getter plugin for Google Cloud Storage\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s <certFile> <keyFile> <caFile> <url>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nThis binary is intended to be called by Helm 4, not directly.\n")
		fmt.Fprintf(os.Stderr, "Install as a Helm plugin and use: helm repo add myrepo gs://bucket/path\n")
		os.Exit(1)
	}

	href := os.Args[4]

	if !strings.HasPrefix(href, "gs://") && !strings.HasPrefix(href, "gcs://") {
		fmt.Fprintf(os.Stderr, "Error: unsupported protocol in URL %q, expected gs:// or gcs://\n", href)
		os.Exit(1)
	}

	serviceAccount := os.Getenv("HELM_GCS_SERVICE_ACCOUNT")

	if err := download(href, serviceAccount); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func download(href, serviceAccount string) error {
	client, err := gcs.NewClient(serviceAccount)
	if err != nil {
		return fmt.Errorf("failed to create GCS client: %w", err)
	}
	defer client.Close()

	obj, err := gcs.Object(client, href)
	if err != nil {
		return fmt.Errorf("failed to get object handle for %q: %w", href, err)
	}

	reader, err := obj.NewReader(context.Background())
	if err != nil {
		return fmt.Errorf("failed to read from GCS: %w", err)
	}
	defer reader.Close()

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return fmt.Errorf("failed to write to stdout: %w", err)
	}

	return nil
}
