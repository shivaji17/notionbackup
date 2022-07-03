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
	baseDirPath        string
	databaseDirPath    string
	pageDirPath        string
	blockDirPath       string
	dataIdentifierList []DataIdentifier
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
		baseDirPath:        basePath,
		databaseDirPath:    databaseDirPath,
		pageDirPath:        pageDirPath,
		blockDirPath:       blockDirPath,
		dataIdentifierList: make([]DataIdentifier, 0),
	}, nil
}

func (rw *FileReaderWriter) writeData(ctx context.Context, v interface{},
	dirPath string) (DataIdentifier, error) {
	dataIdentifier := filepath.Join(dirPath, uuid.New().String())
	dataBytes, err := json.Marshal(&v)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(dataIdentifier, dataBytes, OBJECT_FILE_PERM)
	if err != nil {
		return "", err
	}

	rw.dataIdentifierList = append(rw.dataIdentifierList,
		DataIdentifier(dataIdentifier))
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
	err := rw.readData(ctx, string(identifier), &database)
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
	err := rw.readData(ctx, string(identifier), &page)
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
	databytes, err := os.ReadFile(string(identifier))
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

	for _, identifier := range rw.dataIdentifierList {
		err := os.Remove(identifier.String())
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
