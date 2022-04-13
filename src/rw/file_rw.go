package rw

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/jomei/notionapi"
	"github.com/sawantshivaji1997/notionbackup/src/utils"
)

const (
	DATABASE_DIR_NAME = "databases"
	PAGE_DIR_NAME     = "pages"
	BLOCK_DIR_NAME    = "blocks"
	FILE_PERM         = 0400
)

type FileReaderWriter struct {
	baseDirPath     string
	databaseDirPath string
	pageDirPath     string
	blockDirPath    string
}

func GetFileReaderWriter(basePath string, createDirIfNotExist bool) (ReaderWriter, error) {
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
	pageDirPath := filepath.Join(basePath, PAGE_DIR_NAME)
	blockDirPath := filepath.Join(basePath, BLOCK_DIR_NAME)

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
	}, nil
}

func (rw *FileReaderWriter) writeData(ctx context.Context, dataBytes []byte, dirPath string) (DataIdentifier, error) {
	dataIdentifier := filepath.Join(dirPath, uuid.New().String())
	err := os.WriteFile(dataIdentifier, dataBytes, FILE_PERM)

	if err != nil {
		return "", err
	}

	return DataIdentifier(dataIdentifier), nil
}

func (rw *FileReaderWriter) readData(ctx context.Context, filePath string, v interface{}) error {
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

func (rw *FileReaderWriter) WriteDatabase(ctx context.Context, database *notionapi.Database) (DataIdentifier, error) {
	if database == nil {
		return "", errors.New("nullptr received for database object")
	}

	dataBytes, err := json.Marshal(&database)
	if err != nil {
		return "", err
	}
	return rw.writeData(ctx, dataBytes, rw.databaseDirPath)
}

func (rw *FileReaderWriter) ReadDatabase(ctx context.Context, identifier DataIdentifier) (*notionapi.Database, error) {
	database := &notionapi.Database{}
	err := rw.readData(ctx, string(identifier), &database)
	if err != nil {
		return nil, err
	}
	return database, nil
}

func (rw *FileReaderWriter) WritePage(ctx context.Context, page *notionapi.Page) (DataIdentifier, error) {
	if page == nil {
		return "", errors.New("nullptr received for page object")
	}

	dataBytes, err := json.Marshal(&page)
	if err != nil {
		return "", err
	}
	return rw.writeData(ctx, dataBytes, rw.pageDirPath)
}

func (rw *FileReaderWriter) ReadPage(ctx context.Context, identifier DataIdentifier) (*notionapi.Page, error) {
	page := &notionapi.Page{}
	err := rw.readData(ctx, string(identifier), &page)
	if err != nil {
		return nil, err
	}
	return page, nil
}

func (rw *FileReaderWriter) WriteBlock(ctx context.Context, block notionapi.Block) (DataIdentifier, error) {
	if block == nil {
		return "", errors.New("nullptr received for block object")
	}

	dataBytes, err := json.Marshal(&block)
	if err != nil {
		return "", err
	}
	return rw.writeData(ctx, dataBytes, rw.blockDirPath)
}

func (rw *FileReaderWriter) ReadBlock(ctx context.Context, identifier DataIdentifier) (notionapi.Block, error) {
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
