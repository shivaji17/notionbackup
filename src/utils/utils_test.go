package utils_test

import (
	"testing"

	"github.com/sawantshivaji1997/notionbackup/src/utils"
	"github.com/stretchr/testify/assert"
)

const (
	TESTDATAPATH         = "./../../testdata/"
	FILEPATH             = TESTDATAPATH + "notionclient/search/empty_search_result.json"
	INVALID_FILEPATH     = TESTDATAPATH + "notionclient/search"
	PAGE_JSON            = TESTDATAPATH + "notionclient/page/page.json"
	DATABASE_JSON        = TESTDATAPATH + "notionclient/database/database.json"
	SEARCH_RESPONSE_JSON = TESTDATAPATH + "notionclient/search/search_all_databases.json"
	INVALID_JSON         = TESTDATAPATH + "invalid_json.json"
)

func TestReadContentsOfFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		wantErr  bool
		contents string
	}{
		{
			name:     "Valid file path",
			filePath: FILEPATH,
			wantErr:  false,
			contents: `{
  "object": "list",
  "results": [],
  "next_cursor": null,
  "has_more": false,
  "type": "page_or_database",
  "page_or_database": {}
}`,
		},
		{
			name:     "Invalid file path",
			filePath: INVALID_FILEPATH,
			wantErr:  true,
			contents: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fileData, err := utils.ReadContentsOfFile(test.filePath)

			if test.wantErr {
				assert.Nil(t, fileData)
				assert.NotNil(t, err)
			} else {
				assert.Equal(t, test.contents, string(fileData))
				assert.Nil(t, err)
			}
		})
	}
}

func TestParsePageJsonString(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "Valid Page JSON",
			filePath: PAGE_JSON,
			wantErr:  false,
		},
		{
			name:     "Invalid Page JSON",
			filePath: INVALID_JSON,
			wantErr:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := utils.ReadContentsOfFile(test.filePath)
			assert.Nil(t, err)
			page, err := utils.ParsePageJsonString(data)

			if test.wantErr {
				assert.Nil(t, page)
				assert.NotNil(t, err)
			} else {
				assert.NotNil(t, page)
				assert.Nil(t, err)
			}
		})
	}
}

func TestParseDatabaseJsonString(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "Valid Database JSON",
			filePath: DATABASE_JSON,
			wantErr:  false,
		},
		{
			name:     "Invalid Database JSON",
			filePath: INVALID_JSON,
			wantErr:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := utils.ReadContentsOfFile(test.filePath)
			assert.Nil(t, err)
			database, err := utils.ParseDatabaseJsonString(data)

			if test.wantErr {
				assert.Nil(t, database)
				assert.NotNil(t, err)
			} else {
				assert.NotNil(t, database)
				assert.Nil(t, err)
			}
		})
	}
}

func TestParseSearchResponseJsonString(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "Valid Search response JSON",
			filePath: SEARCH_RESPONSE_JSON,
			wantErr:  false,
		},
		{
			name:     "Invalid Search response JSON",
			filePath: INVALID_JSON,
			wantErr:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := utils.ReadContentsOfFile(test.filePath)
			assert.Nil(t, err)
			database, err := utils.ParseDatabaseJsonString(data)

			if test.wantErr {
				assert.Nil(t, database)
				assert.NotNil(t, err)
			} else {
				assert.NotNil(t, database)
				assert.Nil(t, err)
			}
		})
	}
}
