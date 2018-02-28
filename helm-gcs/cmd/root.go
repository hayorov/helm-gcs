package cmd

import (
	"fmt"
	"os"

	"github.com/nouney/helm-gcs/helm-gcs/gcs"
	"github.com/nouney/helm-gcs/helm-gcs/helm"
	"github.com/nouney/helm-gcs/helm-gcs/repo"
	"github.com/spf13/cobra"
)

var debug bool
var RootCmd = &cobra.Command{
	Use:   "helm-gcs",
	Short: "Manage Helm repositories on Google Cloud Storage",
	Long:  ``,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(func() {
		repo.Debug = debug
		gcs.Debug = debug
		helm.Debug = debug
	})
	RootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "debug mode")
}
