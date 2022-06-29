package config

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/jomei/notionapi"
	"github.com/sawantshivaji1997/notionbackup/src/exporter"
	"github.com/sawantshivaji1997/notionbackup/src/notionclient"
	"github.com/sawantshivaji1997/notionbackup/src/rw"
	"github.com/sawantshivaji1997/notionbackup/src/tree/builder"
)

type OperationType int

const (
	UNKNOWN OperationType = 0
	BACKUP                = 1
	RESTORE               = 2
)

type Config struct {
	Token          string
	Operation_Type OperationType
	PageUUIDs      []string
	DatabaseUUIDs  []string
	Dir            string
	Create_Dir     bool
}

func validateUUIDs(objectType string, uuidList []string) error {
	for _, objectUUID := range uuidList {
		if _, err := uuid.Parse(objectUUID); err != nil {
			return errors.New("invalid " + objectType + " UUID: " + objectUUID)
		}
	}
	return nil
}

func (c *Config) validateBackupConfig() error {
	if c.Token == "" {
		return errors.New("notion secret token not provided")
	}

	if c.Dir == "" {
		c.Dir = "./"
	}

	dir, err := filepath.Abs(c.Dir)
	if err != nil {
		return err
	}
	c.Dir = dir

	err = validateUUIDs("Page", c.PageUUIDs)
	if err != nil {
		return err
	}

	err = validateUUIDs("Database", c.DatabaseUUIDs)
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) executeBackup(ctx context.Context) error {
	ntnClient := notionclient.GetNotionApiClient(ctx, notionapi.Token(c.Token),
		notionapi.NewClient)
	readerWriter, err := rw.GetFileReaderWriter(c.Dir, c.Create_Dir)
	if err != nil {
		return err
	}

	treeBuilderReq := &builder.TreeBuilderRequest{
		PageIdList:     c.PageUUIDs,
		DatabaseIdList: c.DatabaseUUIDs,
	}

	treeBuilder := builder.GetExportTreebuilder(ctx, ntnClient, readerWriter,
		treeBuilderReq)
	tree, err := treeBuilder.BuildTree(ctx)
	if err != nil {
		return err
	}

	err = exporter.ExportTree(ctx, readerWriter, tree)
	return err
}

func (c *Config) Execute(ctx context.Context) error {
	if c.Operation_Type == BACKUP {
		err := c.validateBackupConfig()
		if err != nil {
			return err
		}

		return c.executeBackup(ctx)
	}

	return errors.New("unknown operation type provided")
}
