package cmd

import (
	"github.com/nouney/helm-gcs/helm-gcs/repo"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [gs://bucket/path]",
	Short: "Init a repository",
	Long:  ``,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return repo.CreateRepo(args[0])
	},
}

func init() {
	RootCmd.AddCommand(initCmd)
}
