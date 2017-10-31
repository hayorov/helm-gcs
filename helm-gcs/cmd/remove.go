package cmd

import (
	"strings"

	"github.com/nouney/helm-gcs/helm-gcs/repo"
	"github.com/spf13/cobra"
)

var chartVersion string

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove [chart] [repository]",
	Short: "Remove a chart from a GCS repository",
	Long:  ``,
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		var repoName string
		var chart string
		if len(args) >= 2 {
			chart = args[0]
			repoName = args[1]
		} else if len(args) == 1 {
			chunks := strings.Split(args[0], "/")
			chart = chunks[1]
			repoName = chunks[0]
		}
		r, err := repo.LoadRepo(repoName)
		if err != nil {
			return err
		}
		return r.RemoveChart(chart, chartVersion)
	},
}

func init() {
	RootCmd.AddCommand(removeCmd)
	removeCmd.Flags().StringVarP(&chartVersion, "version", "v", "", "-v <version>")
}
