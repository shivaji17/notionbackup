package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/shivaji17/notionbackup/src/config"
	"github.com/spf13/cobra"
)

var metadataFilePath string
var restoreToPageUUID string

// restoreCmd represents the restore command
var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore the data back to Notion",
	Long: "Restore the data back to Notion with all Pages, Databases and " +
		"Blocks maintaining the hierarchy of all the objects.",
	RunE: Restore,
}

func init() {
	rootCmd.AddCommand(restoreCmd)

	// Here you will define your flags and configuration settings.
	restoreCmd.Flags().StringVarP(&metadataFilePath, "file-path", "f", "",
		"metadata file path")
	restoreCmd.Flags().StringVarP(&restoreToPageUUID, "page", "p", "",
		"page uuid to which all data needs to be restored")
}

func Restore(cmd *cobra.Command, args []string) error {
	validateNonEmptyNotionToken()

	log, err := getLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		return err
	}
	cfg := &config.Config{
		Token:             notionToken,
		Operation_Type:    config.RESTORE,
		MetadataFilePath:  metadataFilePath,
		RestoreToPageUUID: restoreToPageUUID,
	}

	ctx := log.WithContext(context.Background())

	cfg.Execute(ctx, config.InitializeRestore)
	return nil
}
