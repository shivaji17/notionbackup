package config_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jomei/notionapi"
	"github.com/sawantshivaji1997/notionbackup/src/config"
	"github.com/sawantshivaji1997/notionbackup/src/metadata"
	"github.com/sawantshivaji1997/notionbackup/src/mocks"
	"github.com/sawantshivaji1997/notionbackup/src/rw"
	"github.com/sawantshivaji1997/notionbackup/src/tree"
	"github.com/sawantshivaji1997/notionbackup/src/tree/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	MOCKED_TOKEN                  = "mocked_token"
	TESTDATADIR                   = "./../../testdata/"
	TESTDATAPATH                  = TESTDATADIR + "rw"
	INVALID_FILE_PATH             = TESTDATADIR + "file.pb"
	NON_EXISTING_DIR              = TESTDATADIR + "non_existing_dir"
	METADATA_FILEPATH             = TESTDATADIR + "importer/metadata.pb"
	INVALID_METADATA_FILE_CONTENT = TESTDATADIR + "importer/invalid_metadata.pb"
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

func TestInitializeBackup(t *testing.T) {
	t.Run("Valid export data path", func(t *testing.T) {
		cfg := &config.Config{
			Token:          MOCKED_TOKEN,
			Operation_Type: config.BACKUP,
			Dir:            TESTDATAPATH,
			Create_Dir:     false,
		}

		config.InitializeBackup(context.Background(), cfg)

		assert.NotNil(t, cfg.NotionClient)
		assert.NotNil(t, cfg.ReaderWriter)
		assert.NotNil(t, cfg.TreeBuilder)
	})

	t.Run("Directory does not exist", func(t *testing.T) {
		defer func() {
			r := recover()
			assert.NotNilf(t, r, "Panic Recovering")
		}()
		cfg := &config.Config{
			Token:          MOCKED_TOKEN,
			Operation_Type: config.BACKUP,
			Dir:            NON_EXISTING_DIR,
			Create_Dir:     false,
		}

		config.InitializeBackup(context.Background(), cfg)
	})
}

func TestInitializeRestore(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		shouldPanic bool
	}{
		{
			name: "All fields are valid",
			cfg: &config.Config{
				Token:            MOCKED_TOKEN,
				Operation_Type:   config.RESTORE,
				MetadataFilePath: METADATA_FILEPATH,
			},
			shouldPanic: false,
		},
		{
			name: "Metadata file does not exists",
			cfg: &config.Config{
				Token:            MOCKED_TOKEN,
				Operation_Type:   config.RESTORE,
				MetadataFilePath: INVALID_FILE_PATH,
			},
			shouldPanic: true,
		},
		{
			name: "Invalid metadata file data",
			cfg: &config.Config{
				Token:            MOCKED_TOKEN,
				Operation_Type:   config.RESTORE,
				MetadataFilePath: INVALID_METADATA_FILE_CONTENT,
			},
			shouldPanic: true,
		},
	}

	for _, test := range tests {
		if test.shouldPanic {
			t.Run(test.name, func(t *testing.T) {
				defer func() {
					r := recover()
					assert.NotNilf(t, r, "Panic Recovering")
				}()

				config.InitializeRestore(context.Background(), test.cfg)
			})
		} else {
			t.Run(test.name, func(t *testing.T) {
				config.InitializeRestore(context.Background(), test.cfg)
				assert.NotNil(t, test.cfg.NotionClient)
				assert.NotNil(t, test.cfg.ReaderWriter)
				assert.NotNil(t, test.cfg.TreeBuilder)
			})
		}
	}
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

	t.Run("BACKUP: Invalid config: empty token", func(t *testing.T) {
		config := &config.Config{
			Token:          "",
			Operation_Type: config.BACKUP,
		}
		err := config.Execute(context.Background())
		assert.NotNil(err)
	})

	t.Run("BACKUP: Invalid config: invalid page UUID", func(t *testing.T) {
		config := &config.Config{
			Operation_Type: config.BACKUP,
			Token:          MOCKED_TOKEN,
			Dir:            "",
			PageUUIDs:      []string{"05034203-2870-4bc8-b1f9-22c0ae6e56b"},
		}

		err := config.Execute(context.Background())
		assert.NotNil(err)
	})

	t.Run("BACKUP: Invalid config: invalid database UUID", func(t *testing.T) {
		config := &config.Config{
			Operation_Type: config.BACKUP,
			Token:          MOCKED_TOKEN,
			Dir:            "",
			DatabaseUUIDs:  []string{"05034203-2870-4bc8-b1f9-22c0ae6e56b"},
		}

		err := config.Execute(context.Background())
		assert.NotNil(err)
	})

	t.Run("BACKUP: Error while building tree", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)
		mockedTreeBuilder := mocks.NewTreeBuilder(t)

		mockedTreeBuilder.On("BuildTree", context.Background()).Return(
			nil, errGeneric)

		config := &config.Config{
			Token:          MOCKED_TOKEN,
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

	t.Run("BACKUP: Error while writing metadata", func(t *testing.T) {
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
			Token:          MOCKED_TOKEN,
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

	t.Run("BACKUP: Error while cleanup", func(t *testing.T) {
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
			Token:          MOCKED_TOKEN,
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

	t.Run("BACKUP: Valid config", func(t *testing.T) {
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
			Token:          MOCKED_TOKEN,
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

	t.Run("RESTORE: Invalid config: empty token", func(t *testing.T) {
		config := &config.Config{
			Token:          "",
			Operation_Type: config.RESTORE,
		}
		err := config.Execute(context.Background())
		assert.NotNil(err)
	})

	t.Run("RESTORE: Error while building tree", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)
		mockedTreeBuilder := mocks.NewTreeBuilder(t)

		mockedTreeBuilder.On("BuildTree", context.Background()).Return(
			nil, errGeneric)

		config := &config.Config{
			Token:            MOCKED_TOKEN,
			Operation_Type:   config.RESTORE,
			MetadataFilePath: METADATA_FILEPATH,
		}

		ctx := context.Background()
		err := config.Execute(ctx,
			getAssignMockedNotionClientFunc(ctx, mockedNotionClient),
			getAssignMockedRWFunc(ctx, mockedRW),
			getAssignMockedTreeBuilderFunc(ctx, mockedTreeBuilder))

		assert.NotNil(err)
	})

	t.Run("RESTORE: Error while importing objects", func(t *testing.T) {
		ctx := context.Background()
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)
		mockedTreeBuilder := mocks.NewTreeBuilder(t)

		mockedRW.On("WritePage", ctx, mock.Anything).Return(
			rw.DataIdentifier(uuid.NewString()), nil)
		mockedRW.On("ReadPage", ctx, mock.Anything).Return(nil, errGeneric)

		pageNode, err := node.CreatePageNode(ctx, &notionapi.Page{}, mockedRW)
		assert.NotNil(pageNode)
		assert.Nil(err)
		rootNode := node.CreateRootNode()
		assert.NotNil(rootNode)

		rootNode.AddChild(pageNode)

		mockedTreeBuilder.On("BuildTree", context.Background()).Return(
			&tree.Tree{RootNode: rootNode}, nil)

		config := &config.Config{
			Token:            MOCKED_TOKEN,
			Operation_Type:   config.RESTORE,
			MetadataFilePath: METADATA_FILEPATH,
		}

		err = config.Execute(ctx,
			getAssignMockedNotionClientFunc(ctx, mockedNotionClient),
			getAssignMockedRWFunc(ctx, mockedRW),
			getAssignMockedTreeBuilderFunc(ctx, mockedTreeBuilder))

		assert.NotNil(err)
	})

	t.Run("RESTORE: Valid config", func(t *testing.T) {
		ctx := context.Background()
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)
		mockedTreeBuilder := mocks.NewTreeBuilder(t)

		mockedTreeBuilder.On("BuildTree", ctx).Return(
			&tree.Tree{RootNode: node.CreateRootNode()}, nil)

		config := &config.Config{
			Token:            MOCKED_TOKEN,
			Operation_Type:   config.RESTORE,
			MetadataFilePath: METADATA_FILEPATH,
		}

		err := config.Execute(ctx,
			getAssignMockedNotionClientFunc(ctx, mockedNotionClient),
			getAssignMockedRWFunc(ctx, mockedRW),
			getAssignMockedTreeBuilderFunc(ctx, mockedTreeBuilder))

		assert.Nil(err)
	})
}
