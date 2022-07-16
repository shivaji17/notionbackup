package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/jomei/notionapi"
	"github.com/rs/zerolog"
	"github.com/sawantshivaji1997/notionbackup/src/exporter"
	"github.com/sawantshivaji1997/notionbackup/src/logging"
	"github.com/sawantshivaji1997/notionbackup/src/notionclient"
	"github.com/sawantshivaji1997/notionbackup/src/rw"
	"github.com/sawantshivaji1997/notionbackup/src/tree/builder"
)

type OperationType string

const (
	UNKNOWN OperationType = "UNKNOWN"
	BACKUP  OperationType = "BACKUP"
	RESTORE OperationType = "RESTORE"
)

type ConfigOption func(context.Context, *Config)

func Initialize(ctx context.Context, c *Config) {
	log := zerolog.Ctx(ctx)
	var err error
	c.ReaderWriter, err = rw.GetFileReaderWriter(ctx, c.Dir, c.Create_Dir)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create ReaderWriter instance")
		os.Exit(1)
	}

	c.NotionClient = notionclient.GetNotionApiClient(ctx,
		notionapi.Token(c.Token), notionapi.NewClient)

	treeBuilderReq := &builder.TreeBuilderRequest{
		PageIdList:     c.PageUUIDs,
		DatabaseIdList: c.DatabaseUUIDs,
	}

	c.TreeBuilder = builder.GetExportTreebuilder(ctx, c.NotionClient,
		c.ReaderWriter, treeBuilderReq)
}

type Config struct {
	Token          string
	Operation_Type OperationType
	PageUUIDs      []string
	DatabaseUUIDs  []string
	Dir            string
	Create_Dir     bool
	NotionClient   notionclient.NotionClient
	ReaderWriter   rw.ReaderWriter
	TreeBuilder    builder.TreeBuilder
}

func validateUUIDs(objectType string, uuidList []string) error {
	for _, objectUUID := range uuidList {
		if _, err := uuid.Parse(objectUUID); err != nil {
			return fmt.Errorf("invalid %s UUID: %s", objectType, objectUUID)
		}
	}
	return nil
}

func (c *Config) validateBackupConfig() error {
	if c.Token == "" {
		return fmt.Errorf("notion secret token not provided")
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
	log := zerolog.Ctx(ctx)

	tree, err := c.TreeBuilder.BuildTree(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to build the notion object tree")
		return err
	}

	log.Info().Msg("Creating metadata of the exported data")
	err = exporter.ExportTree(ctx, c.ReaderWriter, tree)
	if err != nil {
		log.Error().Err(err).Msg(
			"Failed to create the metadata of the exported data. Cleaning up...")

		err2 := c.ReaderWriter.CleanUp(ctx)
		if err2 != nil {
			log.Warn().Err(err2).Msg(
				"Failed to cleanup the exported data. Manual cleanup may be required")
		} else {
			log.Info().Msg("Cleanup successful")
		}

		return err
	}

	log.Info().Msg("Backup successful")
	return nil
}

func (c *Config) execute(ctx context.Context, opts ...ConfigOption) error {
	log := zerolog.Ctx(ctx)
	if c.Operation_Type == BACKUP {
		err := c.validateBackupConfig()
		if err != nil {
			log.Error().Err(err).Msg(logging.ValidationErr)
			return err
		}

		for _, opt := range opts {
			opt(ctx, c)
		}

		log.Info().Msg("Starting backup operation")

		return c.executeBackup(ctx)
	}

	err := fmt.Errorf("unknown operation type provided: %s", c.Operation_Type)
	log.Error().Err(err).Msg(logging.ValidationErr)
	return err
}

func (c *Config) Execute(ctx context.Context, opts ...ConfigOption) error {
	return c.execute(ctx, opts...)
}
