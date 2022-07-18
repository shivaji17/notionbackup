package config_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/sawantshivaji1997/notionbackup/src/config"
	"github.com/sawantshivaji1997/notionbackup/src/metadata"
	"github.com/sawantshivaji1997/notionbackup/src/mocks"
	"github.com/sawantshivaji1997/notionbackup/src/tree"
	"github.com/sawantshivaji1997/notionbackup/src/tree/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	TESTDATADIR  = "./../../testdata/"
	TESTDATAPATH = TESTDATADIR + "rw/testpath"
)

var errGeneric = fmt.Errorf("generic error")

func getAssignMockedRWFunc(ctx context.Context,
	rw *mocks.ReaderWriter) config.ConfigOption {
	return func(ctx context.Context, c *config.Config) {
		c.ReaderWriter = rw
	}
}

func getAssignMockedNotionClientFunc(ctx context.Context,
	client *mocks.NotionClient) config.
	ConfigOption {
	return func(ctx context.Context, c *config.Config) {
		c.NotionClient = client
	}
}

func getAssignMockedTreeBuilderFunc(ctx context.Context,
	builder *mocks.TreeBuilder) config.
	ConfigOption {
	return func(ctx context.Context, c *config.Config) {
		c.TreeBuilder = builder
	}
}

func TestInitialize(t *testing.T) {
	cfg := &config.Config{
		Token:          "mockedToken",
		Operation_Type: config.BACKUP,
		Dir:            TESTDATAPATH,
		Create_Dir:     false,
	}

	config.Initialize(context.Background(), cfg)

	assert.NotNil(t, cfg.NotionClient)
	assert.NotNil(t, cfg.ReaderWriter)
	assert.NotNil(t, cfg.TreeBuilder)
}

func TestExecute(t *testing.T) {
	assert := assert.New(t)

	t.Run("Invalid operation type", func(t *testing.T) {
		config := &config.Config{
			Operation_Type: config.UNKNOWN,
		}

		err := config.Execute(context.Background())
		assert.NotNil(err)
	})

	t.Run("Invalid config: empty token", func(t *testing.T) {
		config := &config.Config{
			Token:          "",
			Operation_Type: config.BACKUP,
		}
		err := config.Execute(context.Background())
		assert.NotNil(err)
	})

	t.Run("Invalid config: invalid page UUID", func(t *testing.T) {
		config := &config.Config{
			Operation_Type: config.BACKUP,
			Token:          "mockedToken",
			Dir:            "",
			PageUUIDs:      []string{"05034203-2870-4bc8-b1f9-22c0ae6e56b"},
		}

		err := config.Execute(context.Background())
		assert.NotNil(err)
	})

	t.Run("Invalid config: invalid database UUID", func(t *testing.T) {
		config := &config.Config{
			Operation_Type: config.BACKUP,
			Token:          "mockedToken",
			Dir:            "",
			DatabaseUUIDs:  []string{"05034203-2870-4bc8-b1f9-22c0ae6e56b"},
		}

		err := config.Execute(context.Background())
		assert.NotNil(err)
	})

	t.Run("Error while building tree", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)
		mockedTreeBuilder := mocks.NewTreeBuilder(t)

		mockedTreeBuilder.On("BuildTree", context.Background()).Return(
			nil, errGeneric)

		config := &config.Config{
			Token:          "mockedToken",
			Operation_Type: config.BACKUP,
			PageUUIDs: []string{"05034203-2870-4bc8-b1f9-22c0ae6e56ba",
				"53d18605-7779-4700-b16d-662a332283a1"},
			DatabaseUUIDs: []string{"5ed2d97a-510a-4756-b113-cc28c7a30fd7",
				"9cd00ee9-63e5-4dad-b0aa-d76f2ecc36d1"},
			Dir:        TESTDATAPATH,
			Create_Dir: false,
		}

		ctx := context.Background()
		err := config.Execute(ctx,
			getAssignMockedNotionClientFunc(ctx, mockedNotionClient),
			getAssignMockedRWFunc(ctx, mockedRW),
			getAssignMockedTreeBuilderFunc(ctx, mockedTreeBuilder))

		assert.NotNil(err)
	})

	t.Run("Error while writing metadata", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)
		mockedTreeBuilder := mocks.NewTreeBuilder(t)

		mockedTreeBuilder.On("BuildTree", context.Background()).Return(
			&tree.Tree{
				RootNode: node.CreateRootNode(),
			}, nil)

		mockedRW.On("GetStorageConfig", context.Background()).Return(
			&metadata.StorageConfig{}, nil)
		mockedRW.On("WriteMetaData", context.Background(), mock.Anything).Return(
			errGeneric)

		mockedRW.On("CleanUp", context.Background()).Return(nil)

		config := &config.Config{
			Token:          "mockedToken",
			Operation_Type: config.BACKUP,
			PageUUIDs: []string{"05034203-2870-4bc8-b1f9-22c0ae6e56ba",
				"53d18605-7779-4700-b16d-662a332283a1"},
			DatabaseUUIDs: []string{"5ed2d97a-510a-4756-b113-cc28c7a30fd7",
				"9cd00ee9-63e5-4dad-b0aa-d76f2ecc36d1"},
			Dir:        TESTDATAPATH,
			Create_Dir: false,
		}

		ctx := context.Background()
		err := config.Execute(ctx,
			getAssignMockedNotionClientFunc(ctx, mockedNotionClient),
			getAssignMockedRWFunc(ctx, mockedRW),
			getAssignMockedTreeBuilderFunc(ctx, mockedTreeBuilder))

		assert.NotNil(err)
	})

	t.Run("Error while cleanup", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)
		mockedTreeBuilder := mocks.NewTreeBuilder(t)

		mockedTreeBuilder.On("BuildTree", context.Background()).Return(
			&tree.Tree{
				RootNode: node.CreateRootNode(),
			}, nil)

		mockedRW.On("GetStorageConfig", context.Background()).Return(
			&metadata.StorageConfig{}, nil)

		mockedRW.On("WriteMetaData", context.Background(), mock.Anything).Return(
			errGeneric)

		mockedRW.On("CleanUp", context.Background()).Return(errGeneric)

		config := &config.Config{
			Token:          "mockedToken",
			Operation_Type: config.BACKUP,
			PageUUIDs: []string{"05034203-2870-4bc8-b1f9-22c0ae6e56ba",
				"53d18605-7779-4700-b16d-662a332283a1"},
			DatabaseUUIDs: []string{"5ed2d97a-510a-4756-b113-cc28c7a30fd7",
				"9cd00ee9-63e5-4dad-b0aa-d76f2ecc36d1"},
			Dir:        TESTDATAPATH,
			Create_Dir: false,
		}

		ctx := context.Background()
		err := config.Execute(ctx,
			getAssignMockedNotionClientFunc(ctx, mockedNotionClient),
			getAssignMockedRWFunc(ctx, mockedRW),
			getAssignMockedTreeBuilderFunc(ctx, mockedTreeBuilder))

		assert.NotNil(err)
	})

	t.Run("Valid config", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)
		mockedTreeBuilder := mocks.NewTreeBuilder(t)

		mockedRW.On("WriteMetaData", context.Background(), mock.Anything).Return(
			nil)
		mockedRW.On("GetStorageConfig", context.Background()).Return(
			&metadata.StorageConfig{}, nil)

		mockedTreeBuilder.On("BuildTree", context.Background()).Return(
			&tree.Tree{
				RootNode: node.CreateRootNode(),
			}, nil)

		config := &config.Config{
			Token:          "mockedToken",
			Operation_Type: config.BACKUP,
			PageUUIDs: []string{"05034203-2870-4bc8-b1f9-22c0ae6e56ba",
				"53d18605-7779-4700-b16d-662a332283a1"},
			DatabaseUUIDs: []string{"5ed2d97a-510a-4756-b113-cc28c7a30fd7",
				"9cd00ee9-63e5-4dad-b0aa-d76f2ecc36d1"},
			Dir:        TESTDATAPATH,
			Create_Dir: false,
		}

		ctx := context.Background()
		err := config.Execute(ctx,
			getAssignMockedNotionClientFunc(ctx, mockedNotionClient),
			getAssignMockedRWFunc(ctx, mockedRW),
			getAssignMockedTreeBuilderFunc(ctx, mockedTreeBuilder))

		assert.Nil(err)
	})
}
