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
	"github.com/hayorov/helm-gcs/pkg/repo"
	"github.com/spf13/cobra"
)

var (
	flagForce      bool
	flagRetry      bool
	flagPublic     bool
	flagPublicURL  string
	flagBucketPath string
	flagMetadata   map[string]string
)

var pushCmd = &cobra.Command{
	Use:   "push [chart.tar.gz] [repository]",
	Short: "push a chart into a repository",
	Long:  `This command pushes a chart into a repository that has been added to helm via "helm repo add".`,
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		chartpath, repoName := args[0], args[1]
		r, err := repo.Load(repoName, gcsClient)
		if err != nil {
			return err
		}
		return r.PushChart(chartpath, flagForce, flagRetry, flagPublic, flagPublicURL, flagBucketPath, flagMetadata)
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().BoolVar(&flagForce, "force", false, "upload the chart even if already indexed")
	pushCmd.Flags().BoolVar(&flagRetry, "retry", false, "retry if the index changed")
	pushCmd.Flags().BoolVar(&flagPublic, "public", false, "expose HTTP URL instead of default gs:// for public buckets")
	pushCmd.Flags().StringVar(&flagPublicURL, "publicUrl", "", "used with --public to overwrite google storage default url")
	pushCmd.Flags().StringVar(&flagBucketPath, "bucketPath", "", "path inside the google bucket")
	pushCmd.Flags().StringToStringVar(&flagMetadata, "metadata", nil, "comma seperated object metadata in the form of key=value")
}
