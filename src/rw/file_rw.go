package rw

import (
	"path/filepath"

	"github.com/jomei/notionapi"
	"github.com/sawantshivaji1997/notionbackup/src/utils"
)

const (
	DATABASE_DIR_NAME = "databases"
	PAGE_DIR_NAME     = "pages"
	BLOCK_DIR_NAME    = "blocks"
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

func (rw *FileReaderWriter) WriteDatabase(database *notionapi.Database) (DataIdentifier, error) {
	return "", nil
}

func (rw *FileReaderWriter) ReadDatabase(identifier DataIdentifier) (*notionapi.Database, error) {
	return nil, nil
}

func (rw *FileReaderWriter) WritePage(page *notionapi.Page) (DataIdentifier, error) {
	return "", nil
}

func (rw *FileReaderWriter) ReadPage(identifier DataIdentifier) (*notionapi.Page, error) {
	return nil, nil
}

func (rw *FileReaderWriter) WriteBlock(block notionapi.Block) (DataIdentifier, error) {
	return "", nil
}

func (rw *FileReaderWriter) ReadBlock(identifier DataIdentifier) (notionapi.Block, error) {
	return nil, nil
}
