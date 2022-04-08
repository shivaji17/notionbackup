package utils

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/jomei/notionapi"
)

func ReadJsonFile(filePath string) ([]byte, error) {
	jsonFile, err := os.Open(filePath)

	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}
	return byteValue, nil
}

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
