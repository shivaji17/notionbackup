package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var pageUUIDs []string
var databaseUUIDs []string
var backupWorkspace bool

// backupCmd represents the backup command
var backupCmd = &cobra.Command{
	Use: "backup",
	Short: "Take backup of the whole Notion workspace or for the given Notion " +
		"object",
	Long: "Take backup of whole Notion workspace or a specific set of Pages or " +
		"Databases orcombination of both.",
}

func init() {
	rootCmd.AddCommand(backupCmd)

	rootCmd.CompletionOptions.DisableDefaultCmd = true
	backupCmd.PersistentFlags().BoolVarP(&backupWorkspace, "workspace", "w",
		false, "backup whole workspace")

	backupCmd.PersistentFlags().StringArrayVar(&pageUUIDs, "page",
		make([]string, 0), "Page UUIDs for which backup needs to be taken")

	backupCmd.PersistentFlags().StringArrayVar(&databaseUUIDs, "database",
		make([]string, 0), "Database UUIDs for which backup needs to be taken")
}

func validateMutuallyExclusiveFlags() {
	if len(pageUUIDs) == 0 && len(databaseUUIDs) == 0 && !backupWorkspace {
		fmt.Fprintf(os.Stderr, "Please provide --workspace flag to backup whole "+
			"workspace or Page and/or Database UUIDs to backup.\n")
		os.Exit(1)
	}

	if (len(pageUUIDs) != 0 || len(databaseUUIDs) != 0) && backupWorkspace {
		fmt.Fprintf(os.Stderr, "Flag --workspace is mutually exclusive with flag "+
			"--page and --database.\n")
		os.Exit(1)
	}
}
