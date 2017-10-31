package cmd

import (
	"io"
	"log"
	"os"

	"github.com/nouney/helm-gcs/helm-gcs/gcs"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull [file]",
	Short: "pull a file from GCS",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		r, err := gcs.NewReader(args[0])
		if err != nil {
			log.Panic(err)
		}
		_, err = io.Copy(os.Stdout, r)
		return err
	},
}

func init() {
	RootCmd.AddCommand(pullCmd)
}
