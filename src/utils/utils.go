package utils

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/jomei/notionapi"
)

func ParsePageJsonString(jsonBytes []byte) (*notionapi.Page, error) {
	page := &notionapi.Page{}
	err := json.Unmarshal(jsonBytes, &page)
	if err != nil {
		return nil, err
	}
	return page, nil
}

func ParseSearchResponseJsonString(jsonBytes []byte) (*notionapi.SearchResponse, error) {
	searchResponse := &notionapi.SearchResponse{}
	err := json.Unmarshal(jsonBytes, &searchResponse)
	if err != nil {
		return nil, err
	}
	return searchResponse, nil
}

func ParseDatabaseJsonString(jsonBytes []byte) (*notionapi.Database, error) {
	database := &notionapi.Database{}
	err := json.Unmarshal(jsonBytes, &database)
	if err != nil {
		return nil, err
	}
	return database, nil
}

func CheckIfDirExists(dirPath string) error {
	_, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		return err
	}
	return nil
}

func CreateDirectory(dirPath string) error {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return err
	}

	return os.MkdirAll(absPath, 0700)
}
