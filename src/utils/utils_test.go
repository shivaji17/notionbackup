package utils_test

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/jomei/notionapi"
	"github.com/shivaji17/notionbackup/src/utils"
	"github.com/stretchr/testify/assert"
)

const (
	TESTDATAPATH         = "./../../testdata/"
	FILEPATH             = TESTDATAPATH + "notionclient/search/empty_search_result.json"
	INVALID_FILEPATH     = TESTDATAPATH + "notionclient/search"
	PAGE_JSON            = TESTDATAPATH + "notionclient/page/page.json"
	DATABASE_JSON        = TESTDATAPATH + "notionclient/database/database.json"
	PAGE_BLOCKS_JSON     = TESTDATAPATH + "notionclient/block/page_blocks.json"
	SEARCH_RESPONSE_JSON = TESTDATAPATH + "notionclient/search/search_all_databases.json"
	INVALID_JSON         = TESTDATAPATH + "invalid_json.json"
	EXISTING_DIR         = TESTDATAPATH
	NON_EXISTING_DIR     = TESTDATAPATH + "xyz"
	VALID_DIR_PATH       = TESTDATAPATH + "create_dir_test"
	INVALID_DIR_PATH     = "/xyz/sd/^7$%"
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
			fileData, err := ioutil.ReadFile(test.filePath)

			if test.wantErr {
				assert.Empty(t, fileData)
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
			data, err := ioutil.ReadFile(test.filePath)
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
			data, err := ioutil.ReadFile(test.filePath)
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
			data, err := ioutil.ReadFile(test.filePath)
			assert.Nil(t, err)
			resp, err := utils.ParseSearchResponseJsonString(data)

			if test.wantErr {
				assert.Nil(t, resp)
				assert.NotNil(t, err)
			} else {
				assert.NotNil(t, resp)
				assert.Nil(t, err)
			}
		})
	}
}

func TestCheckIfDirExists(t *testing.T) {
	tests := []struct {
		name    string
		dirPath string
		wantErr bool
	}{
		{
			name:    "Existing directory",
			dirPath: EXISTING_DIR,
			wantErr: false,
		},
		{
			name:    "Non existing directory",
			dirPath: NON_EXISTING_DIR,
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := utils.CheckIfDirExists(test.dirPath)
			if test.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestCreateDirectory(t *testing.T) {
	tests := []struct {
		name    string
		dirPath string
		wantErr bool
	}{
		{
			name:    "Valid dir path",
			dirPath: VALID_DIR_PATH,
			wantErr: false,
		},
		{
			name:    "Invalid dir path",
			dirPath: INVALID_DIR_PATH,
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := utils.CreateDirectory(test.dirPath)
			if test.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestDecodeBlockObject(t *testing.T) {

	jsonBytes, err := ioutil.ReadFile(PAGE_BLOCKS_JSON)
	if err != nil {
		t.Fatal(err)
	}

	childBlocks := &notionapi.GetChildrenResponse{}
	err = json.Unmarshal(jsonBytes, &childBlocks)
	if err != nil {
		t.Fatal(err)
	}

	for _, block := range childBlocks.Results {
		t.Run("Testing block type: "+string(block.GetType()), func(t *testing.T) {
			bytes, err := json.Marshal(block)
			assert.Nil(t, err)
			assert.NotEmpty(t, bytes)

			var response map[string]interface{}
			err = json.Unmarshal(bytes, &response)
			if err != nil {
				t.Fatal(err)
			}

			block, err := utils.DecodeBlockObject(response)
			assert.Nil(t, err)
			assert.NotNil(t, block)
		})
	}
}

func TestGetUniqueValues(t *testing.T) {
	input := []string{"a", "b", "c", "d", "a", "b"}
	expectedOutput := []string{"a", "b", "c", "d"}

	output := utils.GetUniqueValues(input)
	assert.Equal(t, expectedOutput, output)
}
