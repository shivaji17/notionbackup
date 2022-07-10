package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/sawantshivaji1997/notionbackup/src/config"
	"github.com/sawantshivaji1997/notionbackup/src/utils"
	"github.com/spf13/cobra"
)

var dir string
var createDir bool

// localCmd represents the local command
var localCmd = &cobra.Command{
	Use:   "local",
	Short: "backup to local machine",
	RunE:  TakeLocalBackup,
}

func init() {
	backupCmd.AddCommand(localCmd)

	// Here you will define your flags and configuration settings.
	localCmd.Flags().StringVarP(&dir, "dir", "d", "",
		"directory to write backup data to")
	localCmd.MarkFlagDirname("dir")
	localCmd.MarkFlagRequired("dir")
	localCmd.Flags().BoolVar(&createDir, "create-dir", false,
		"Create directory if not exists")
}

func TakeLocalBackup(cmd *cobra.Command, args []string) error {

	validateNonEmptyNotionToken()
	validateMutuallyExclusiveFlags()

	log, err := getLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		return err
	}

	pageUUIDs = utils.GetUniqueValues(pageUUIDs)
	databaseUUIDs = utils.GetUniqueValues(databaseUUIDs)

	cfg := &config.Config{
		Token:          notionToken,
		Operation_Type: config.BACKUP,
		PageUUIDs:      pageUUIDs,
		DatabaseUUIDs:  databaseUUIDs,
		Dir:            dir,
		Create_Dir:     createDir,
	}

	ctx := log.WithContext(context.Background())

	cfg.Execute(ctx)
	return nil
}
