package cmd

import (
	"github.com/nouney/helm-gcs/helm-gcs/repo"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push [chart] [repository]",
	Short: "Push a chart into a GCS repository",
	Long:  ``,
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		path, repoName := args[0], args[1]
		r, err := repo.LoadRepo(repoName)
		if err != nil {
			return err
		}
		err = r.AddChartFile(path)
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(pushCmd)
}
