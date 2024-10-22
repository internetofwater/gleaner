package cli

import (
	"fmt"

	"github.com/gleanerio/gleaner/pkg"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// gleanerCmd represents the run command
var versionCmd = &cobra.Command{
	Use:              "version",
	TraverseChildren: true,
	Short:            "returns version ",
	Long: `returns version
`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("version called")

		fmt.Println("Version: " + pkg.VERSION)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
