// Copyright © 2018 Valentin Tjoncke <valtjo@gmail.com>
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

package cmd

import (
	"fmt"
	"os"

	"cloud.google.com/go/storage"
	"github.com/hayorov/helm-gcs/pkg/gcs"
	"github.com/hayorov/helm-gcs/pkg/repo"
	"github.com/spf13/cobra"
)

var (
	gcsClient *storage.Client

	flagServiceAccount string
	flagDebug          bool
)

var rootCmd = &cobra.Command{
	Use:   "helm-gcs",
	Short: "Manage Helm repositories on Google Cloud Storage",
	Long:  ``,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip GCS client initialization for commands that don't need it
		if cmd.Name() == "version" {
			if flagDebug {
				repo.Debug = true
			}
			return nil
		}

		// Initialize GCS client for other commands
		var err error
		gcsClient, err = gcs.NewClient(flagServiceAccount)
		if err != nil {
			return fmt.Errorf("failed to create GCS client: %w", err)
		}
		if flagDebug {
			repo.Debug = true
		}
		return nil
	},
}

// Execute executes the CLI
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagServiceAccount, "service-account", "", "service account to use for GCS")
	rootCmd.PersistentFlags().BoolVar(&flagDebug, "debug", false, "activate debug")
}
