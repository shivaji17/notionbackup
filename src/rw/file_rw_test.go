package rw_test

import (
	"context"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/jomei/notionapi"
	"github.com/sawantshivaji1997/notionbackup/src/rw"
	"github.com/stretchr/testify/assert"
)

const (
	TESTDATADIR       = "./../../testdata/"
	TESTDATAPATH      = TESTDATADIR + "testpath"
	EXISTING_DIR_PATH = TESTDATAPATH
	NON_EXISTING_DIR  = TESTDATAPATH + "test_directory"
	NON_EXISTING_DIR2 = TESTDATAPATH + "test_directory2"
	INVALID_DIR_PATH  = "/xyz/sd/^7$%"
	NOTION_DATA_DIR   = TESTDATADIR + "notionclient/"
	NON_EXISTING_JSON = TESTDATADIR + "xyz.json"
	DATABASE_JSON     = NOTION_DATA_DIR + "database/database.json"
	PAGE_JSON         = NOTION_DATA_DIR + "page/page.json"
	BLOCK_JSON        = NOTION_DATA_DIR + "block/block.json"
	INVALID_JSON      = TESTDATADIR + "invalid_json.json"
)

func checkFilePermissions(t *testing.T, filePath string) {
	fileInfo, err := os.Stat(filePath)
	assert.Nil(t, err)
	assert.Equal(t, fs.FileMode(0400), fileInfo.Mode())
}

func TestGetFileReaderWriter(t *testing.T) {
	tests := []struct {
		name                string
		baseDirPath         string
		createDirIfNotExist bool
		wantErr             bool
		cleanupRequied      bool
	}{
		{
			name:                "Base directory exists",
			baseDirPath:         EXISTING_DIR_PATH,
			createDirIfNotExist: false,
			wantErr:             false,
			cleanupRequied:      false,
		},
		{
			name:                "Create base directory if not exists",
			baseDirPath:         NON_EXISTING_DIR,
			createDirIfNotExist: true,
			wantErr:             false,
			cleanupRequied:      true,
		},
		{
			name:                "Do not create base directory if not exists",
			baseDirPath:         NON_EXISTING_DIR2,
			createDirIfNotExist: false,
			wantErr:             true,
			cleanupRequied:      true,
		},
		{
			name:                "Invalid directory path",
			baseDirPath:         INVALID_DIR_PATH,
			createDirIfNotExist: false,
			wantErr:             true,
			cleanupRequied:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fileRW, err := rw.GetFileReaderWriter(test.baseDirPath, test.createDirIfNotExist)
			if test.wantErr {
				assert.Nil(t, fileRW)
				assert.NotNil(t, err)
			} else {
				assert.NotNil(t, fileRW)
				assert.Nil(t, err)
			}
			if test.cleanupRequied {
				os.RemoveAll(test.baseDirPath)
			}
		})
	}
}

func TestWriteDatabase(t *testing.T) {
	timestamp, err := time.Parse(time.RFC3339, "2021-05-24T05:06:34.827Z")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name           string
		databaseObj    *notionapi.Database
		wantErr        bool
		cleanupRequied bool
	}{
		{
			name: "Valid Database object",
			databaseObj: &notionapi.Database{
				Object:         notionapi.ObjectTypeDatabase,
				ID:             "some_id",
				CreatedTime:    timestamp,
				LastEditedTime: timestamp,
				Title: []notionapi.RichText{
					{
						Type:        notionapi.ObjectTypeText,
						Text:        notionapi.Text{Content: "Test Database"},
						Annotations: &notionapi.Annotations{Color: "default"},
						PlainText:   "Test Database",
						Href:        "",
					},
				},
			},
			wantErr:        false,
			cleanupRequied: true,
		},
		{
			name:           "Nil Database object",
			databaseObj:    nil,
			wantErr:        true,
			cleanupRequied: false,
		},
	}

	filerw, err := rw.GetFileReaderWriter(TESTDATAPATH, true)
	assert.Nil(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			identifier, err := filerw.WriteDatabase(context.Background(), test.databaseObj)
			if test.wantErr {
				assert.Empty(t, identifier)
				assert.NotNil(t, err)
			} else {
				assert.NotEmpty(t, identifier)
				checkFilePermissions(t, string(identifier))
				assert.Nil(t, err)
			}
			if test.cleanupRequied {
				os.RemoveAll(string(identifier))
			}
		})
	}
}

func TestReadDatabase(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "Valid database data from file",
			filePath: DATABASE_JSON,
			wantErr:  false,
		},
		{
			name:     "Invalid json data",
			filePath: INVALID_JSON,
			wantErr:  true,
		},
		{
			name:     "Non Existing file",
			filePath: NON_EXISTING_JSON,
			wantErr:  true,
		},
	}

	filerw, err := rw.GetFileReaderWriter(TESTDATAPATH, true)
	assert.Nil(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database, err := filerw.ReadDatabase(context.Background(), rw.DataIdentifier(test.filePath))
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

func TestWritePage(t *testing.T) {
	timestamp, err := time.Parse(time.RFC3339, "2021-05-24T05:06:34.827Z")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name           string
		pageObj        *notionapi.Page
		wantErr        bool
		cleanupRequied bool
	}{
		{
			name: "Valid Page object",
			pageObj: &notionapi.Page{
				Object:         notionapi.ObjectTypePage,
				ID:             "some_id",
				CreatedTime:    timestamp,
				LastEditedTime: timestamp,
				Parent: notionapi.Parent{
					Type:       notionapi.ParentTypeDatabaseID,
					DatabaseID: "some_id",
				},
				Archived: false,
				URL:      "some_url",
				Properties: notionapi.Properties{
					"Tags": &notionapi.MultiSelectProperty{
						ID:   ";s|V",
						Type: "multi_select",
						MultiSelect: []notionapi.Option{
							{
								ID:    "some_id",
								Name:  "tag",
								Color: "blue",
							},
						},
					},
					"Some another column": &notionapi.PeopleProperty{
						ID:   "rJt\\",
						Type: "people",
						People: []notionapi.User{
							{
								Object:    "user",
								ID:        "some_id",
								Name:      "some name",
								AvatarURL: "some.url",
								Type:      "person",
								Person: &notionapi.Person{
									Email: "some@email.com",
								},
							},
						},
					},
					"SomeColumn": &notionapi.RichTextProperty{
						ID:   "~j_@",
						Type: "rich_text",
						RichText: []notionapi.RichText{
							{
								Type: "text",
								Text: notionapi.Text{
									Content: "some text",
								},
								Annotations: &notionapi.Annotations{
									Color: "default",
								},
								PlainText: "some text",
							},
						},
					},
					"Name": &notionapi.TitleProperty{
						ID:   "title",
						Type: "title",
						Title: []notionapi.RichText{
							{
								Type: "text",
								Text: notionapi.Text{
									Content: "Hello",
								},
								Annotations: &notionapi.Annotations{
									Color: "default",
								},
								PlainText: "Hello",
							},
						},
					},
					"RollupArray": &notionapi.RollupProperty{
						ID:   "abcd",
						Type: "rollup",
						Rollup: notionapi.Rollup{
							Type: "array",
							Array: notionapi.PropertyArray{
								&notionapi.NumberProperty{
									Type:   "number",
									Number: 42.2,
								},
								&notionapi.NumberProperty{
									Type:   "number",
									Number: 56,
								},
							},
						},
					},
				},
			},
			wantErr:        false,
			cleanupRequied: true,
		},
		{
			name:           "Nil Page object",
			pageObj:        nil,
			wantErr:        true,
			cleanupRequied: false,
		},
	}

	filerw, err := rw.GetFileReaderWriter(TESTDATAPATH, true)
	assert.Nil(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			identifier, err := filerw.WritePage(context.Background(), test.pageObj)
			if test.wantErr {
				assert.Empty(t, identifier)
				assert.NotNil(t, err)
			} else {
				assert.NotEmpty(t, identifier)
				checkFilePermissions(t, string(identifier))
				assert.Nil(t, err)
			}
			if test.cleanupRequied {
				os.RemoveAll(string(identifier))
			}
		})
	}

}

func TestReadPage(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "Valid page data from file",
			filePath: PAGE_JSON,
			wantErr:  false,
		},
		{
			name:     "Invalid json data",
			filePath: INVALID_JSON,
			wantErr:  true,
		},
		{
			name:     "Non Existing file",
			filePath: NON_EXISTING_JSON,
			wantErr:  true,
		},
	}

	filerw, err := rw.GetFileReaderWriter(TESTDATAPATH, true)
	assert.Nil(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			page, err := filerw.ReadPage(context.Background(), rw.DataIdentifier(test.filePath))
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

func TestWriteBlock(t *testing.T) {
	timestamp, err := time.Parse(time.RFC3339, "2021-05-24T05:06:34.827Z")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name           string
		blockObj       notionapi.Block
		wantErr        bool
		cleanupRequied bool
	}{
		{
			name: "Valid Block object",
			blockObj: &notionapi.ChildPageBlock{
				BasicBlock: notionapi.BasicBlock{
					Object:         notionapi.ObjectTypeBlock,
					ID:             "some_id",
					Type:           notionapi.BlockTypeChildPage,
					CreatedTime:    &timestamp,
					LastEditedTime: &timestamp,
					HasChildren:    true,
				},
				ChildPage: struct {
					Title string `json:"title"`
				}{
					Title: "Hello",
				},
			},
			wantErr:        false,
			cleanupRequied: true,
		},
		{
			name:           "Nil Block object",
			blockObj:       nil,
			wantErr:        true,
			cleanupRequied: false,
		},
	}

	filerw, err := rw.GetFileReaderWriter(TESTDATAPATH, true)
	assert.Nil(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			identifier, err := filerw.WriteBlock(context.Background(), test.blockObj)
			if test.wantErr {
				assert.Empty(t, identifier)
				assert.NotNil(t, err)
			} else {
				assert.NotEmpty(t, identifier)
				checkFilePermissions(t, string(identifier))
				assert.Nil(t, err)
			}
			if test.cleanupRequied {
				os.RemoveAll(string(identifier))
			}
		})
	}
}

func TestReadBlock(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		wantErr  bool
	}{
		{
			name:     "Valid block data from file",
			filePath: BLOCK_JSON,
			wantErr:  false,
		},
		{
			name:     "Invalid json data",
			filePath: INVALID_JSON,
			wantErr:  true,
		},
		{
			name:     "Non Existing file",
			filePath: NON_EXISTING_JSON,
			wantErr:  true,
		},
	}

	filerw, err := rw.GetFileReaderWriter(TESTDATAPATH, true)
	assert.Nil(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			block, err := filerw.ReadBlock(context.Background(), rw.DataIdentifier(test.filePath))
			if test.wantErr {
				assert.Nil(t, block)
				assert.NotNil(t, err)
			} else {
				assert.NotNil(t, block)
				assert.Nil(t, err)
			}
		})
	}
}
