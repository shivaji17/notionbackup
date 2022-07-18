package rw

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/jomei/notionapi"
	"github.com/rs/zerolog"
	"github.com/sawantshivaji1997/notionbackup/src/logging"
	"github.com/sawantshivaji1997/notionbackup/src/metadata"
	"github.com/sawantshivaji1997/notionbackup/src/utils"
	"google.golang.org/protobuf/proto"
)

const (
	DATABASE_DIR_NAME  = "databases"
	PAGE_DIR_NAME      = "pages"
	BLOCK_DIR_NAME     = "blocks"
	OBJECT_FILE_PERM   = 0400
	METADATA_FILE_PERM = 0644
	METADATA_FILE_NAME = "metadata.pb"
)

type FileReaderWriter struct {
	baseDirPath     string
	databaseDirPath string
	pageDirPath     string
	blockDirPath    string
	filePathList    []string
}

func GetFileReaderWriter(ctx context.Context, basePath string,
	createDirIfNotExist bool) (ReaderWriter, error) {
	log := zerolog.Ctx(ctx)
	err := utils.CheckIfDirExists(basePath)
	if err != nil {
		if !createDirIfNotExist {
			return nil, err
		}

		err = utils.CreateDirectory(basePath)
		if err != nil {
			return nil, err
		}
	}

	databaseDirPath := filepath.Join(basePath, DATABASE_DIR_NAME)
	log.Info().Str(logging.ExportPath, databaseDirPath).Msg(
		"Database objects backup path")

	pageDirPath := filepath.Join(basePath, PAGE_DIR_NAME)
	log.Info().Str(logging.ExportPath, pageDirPath).Msg(
		"Page objects backup path")

	blockDirPath := filepath.Join(basePath, BLOCK_DIR_NAME)
	log.Info().Str(logging.ExportPath, blockDirPath).Msg(
		"Block objects backup path")

	err = utils.CreateDirectory(databaseDirPath)
	if err != nil {
		return nil, err
	}

	err = utils.CreateDirectory(pageDirPath)
	if err != nil {
		return nil, err
	}

	err = utils.CreateDirectory(blockDirPath)
	if err != nil {
		return nil, err
	}

	return &FileReaderWriter{
		baseDirPath:     basePath,
		databaseDirPath: databaseDirPath,
		pageDirPath:     pageDirPath,
		blockDirPath:    blockDirPath,
		filePathList:    make([]string, 0),
	}, nil
}

func GetFileReaderWriterForMetadata(ctx context.Context,
	metadataFilePath string, data *metadata.MetaData) (ReaderWriter, error) {
	baseDir := filepath.Dir(metadataFilePath)
	absBasePath, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, err
	}

	pageDir := filepath.Join(absBasePath, data.StorageConfig.GetLocal().PageDir)
	databaseDir := filepath.Join(absBasePath,
		data.StorageConfig.GetLocal().DatabaseDir)
	blockDir := filepath.Join(absBasePath,
		data.StorageConfig.GetLocal().BlocksDir)

	err = utils.CheckIfDirExists(pageDir)
	if err != nil {
		return nil, err
	}

	err = utils.CheckIfDirExists(databaseDir)
	if err != nil {
		return nil, err
	}

	err = utils.CheckIfDirExists(blockDir)
	if err != nil {
		return nil, err
	}

	return &FileReaderWriter{
		baseDirPath:     absBasePath,
		databaseDirPath: databaseDir,
		pageDirPath:     pageDir,
		blockDirPath:    blockDir,
		filePathList:    make([]string, 0),
	}, nil
}

func (rw *FileReaderWriter) writeData(ctx context.Context, v interface{},
	dirPath string) (DataIdentifier, error) {
	dataIdentifier := uuid.NewString()
	filePath := filepath.Join(dirPath, dataIdentifier)
	dataBytes, err := json.Marshal(&v)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(filePath, dataBytes, OBJECT_FILE_PERM)
	if err != nil {
		return "", err
	}

	rw.filePathList = append(rw.filePathList, filePath)
	return DataIdentifier(dataIdentifier), nil
}

func (rw *FileReaderWriter) readData(ctx context.Context, filePath string,
	v interface{}) error {
	databytes, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(databytes, v)
	if err != nil {
		return err
	}

	return nil
}

func (rw *FileReaderWriter) WriteDatabase(ctx context.Context,
	database *notionapi.Database) (DataIdentifier, error) {
	if database == nil {
		return "", fmt.Errorf("nullptr received for database object")
	}

	return rw.writeData(ctx, database, rw.databaseDirPath)
}

func (rw *FileReaderWriter) ReadDatabase(ctx context.Context,
	identifier DataIdentifier) (*notionapi.Database, error) {
	database := &notionapi.Database{}
	err := rw.readData(ctx, filepath.Join(rw.databaseDirPath,
		identifier.String()), &database)
	if err != nil {
		return nil, err
	}
	return database, nil
}

func (rw *FileReaderWriter) WritePage(ctx context.Context,
	page *notionapi.Page) (DataIdentifier, error) {
	if page == nil {
		return "", fmt.Errorf("nullptr received for page object")
	}

	return rw.writeData(ctx, page, rw.pageDirPath)
}

func (rw *FileReaderWriter) ReadPage(ctx context.Context,
	identifier DataIdentifier) (*notionapi.Page, error) {
	page := &notionapi.Page{}
	err := rw.readData(ctx, filepath.Join(rw.pageDirPath,
		identifier.String()), &page)
	if err != nil {
		return nil, err
	}
	return page, nil
}

func (rw *FileReaderWriter) WriteBlock(ctx context.Context,
	block notionapi.Block) (DataIdentifier, error) {
	if block == nil {
		return "", fmt.Errorf("nullptr received for block object")
	}

	return rw.writeData(ctx, block, rw.blockDirPath)
}

func (rw *FileReaderWriter) ReadBlock(ctx context.Context,
	identifier DataIdentifier) (notionapi.Block, error) {
	databytes, err := os.ReadFile(filepath.Join(rw.blockDirPath,
		identifier.String()))
	if err != nil {
		return nil, err
	}

	var response map[string]interface{}
	err = json.Unmarshal(databytes, &response)
	if err != nil {
		return nil, err
	}

	return utils.DecodeBlockObject(response)
}

func (rw *FileReaderWriter) CleanUp(ctx context.Context) error {
	var externalErr error
	externalErr = nil

	for _, filePath := range rw.filePathList {
		err := os.Remove(filePath)
		if err != nil {
			externalErr = err
		}
	}

	return externalErr
}

func (rw *FileReaderWriter) WriteMetaData(ctx context.Context,
	metadata *metadata.MetaData) error {
	dataBytes, err := proto.Marshal(metadata)
	if err != nil {
		return err
	}

	path := filepath.Join(rw.baseDirPath, METADATA_FILE_NAME)
	zerolog.Ctx(ctx).Info().Str(logging.MetaDataFilePath, path).Msg(
		"Writing Metadata file")
	err = os.WriteFile(path, dataBytes, METADATA_FILE_PERM)
	if err != nil {
		return err
	}

	return nil
}

func (rw *FileReaderWriter) GetStorageConfig(ctx context.Context) (
	*metadata.StorageConfig, error) {
	localConfig := &metadata.StorageConfig_Local{
		DatabaseDir: DATABASE_DIR_NAME,
		PageDir:     PAGE_DIR_NAME,
		BlocksDir:   BLOCK_DIR_NAME,
	}

	return &metadata.StorageConfig{
		Config: &metadata.StorageConfig_Local_{
			Local: localConfig,
		},
	}, nil
}
