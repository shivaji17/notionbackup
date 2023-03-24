package rw_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jomei/notionapi"
	"github.com/shivaji17/notionbackup/src/metadata"
	"github.com/shivaji17/notionbackup/src/rw"
	"github.com/shivaji17/notionbackup/src/utils"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

const (
	TESTDATADIR       = "./../../testdata/"
	RW_DATA_DIR       = TESTDATADIR + "rw/"
	TESTDATAPATH      = RW_DATA_DIR + "testpath"
	EXISTING_DIR_PATH = RW_DATA_DIR
	NON_EXISTING_DIR  = TESTDATAPATH + "test_directory"
	NON_EXISTING_DIR2 = TESTDATAPATH + "test_directory2"
	INVALID_DIR_PATH  = "/xyz/sd/^7$%"
	DATABASE_ID       = "database.json"
	PAGE_ID           = "page.json"
	BLOCK_ID          = "block.json"
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
			fileRW, err := rw.GetFileReaderWriter(context.Background(),
				test.baseDirPath, test.createDirIfNotExist)
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

func createDirs(t *testing.T, baseDir string, pageDir, databaseDir,
	blockDir bool, data *metadata.MetaData) string {
	if pageDir {
		err := utils.CreateDirectory(filepath.Join(baseDir, rw.PAGE_DIR_NAME))
		assert.Nil(t, err)
	}

	if databaseDir {
		err := utils.CreateDirectory(filepath.Join(baseDir, rw.DATABASE_DIR_NAME))
		assert.Nil(t, err)
	}

	if blockDir {
		err := utils.CreateDirectory(filepath.Join(baseDir, rw.BLOCK_DIR_NAME))
		assert.Nil(t, err)
	}
	dataBytes, err := proto.Marshal(data)
	assert.Nil(t, err)

	path := filepath.Join(baseDir, "metadata_test.pb")
	err = os.WriteFile(path, dataBytes, rw.METADATA_FILE_PERM)
	assert.Nil(t, err)
	return path
}

func TestGetFileReaderWriterForMetadata(t *testing.T) {
	storageConfig := &metadata.StorageConfig{
		Config: &metadata.StorageConfig_Local_{
			Local: &metadata.StorageConfig_Local{
				BlocksDir:   rw.BLOCK_DIR_NAME,
				PageDir:     rw.PAGE_DIR_NAME,
				DatabaseDir: rw.DATABASE_DIR_NAME,
			},
		},
	}

	metadataConfig := &metadata.MetaData{
		StorageConfig: storageConfig,
	}

	tests := []struct {
		name        string
		pageDir     bool
		databaseDir bool
		blockDir    bool
		wantErr     bool
	}{
		{
			name:        "Create fileReaderWriter instance successful",
			pageDir:     true,
			databaseDir: true,
			blockDir:    true,
			wantErr:     false,
		},
		{
			name:        "Page directory does not exist",
			pageDir:     false,
			databaseDir: true,
			blockDir:    true,
			wantErr:     true,
		},
		{
			name:        "Database directory does not exist",
			pageDir:     true,
			databaseDir: false,
			blockDir:    true,
			wantErr:     true,
		},
		{
			name:        "Block directory does not exist",
			pageDir:     true,
			databaseDir: true,
			blockDir:    false,
			wantErr:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			baseDir := filepath.Join(TESTDATAPATH, uuid.NewString())
			metadataFilePath := createDirs(t, baseDir, test.pageDir,
				test.databaseDir, test.blockDir, metadataConfig)
			fileRW, err := rw.GetFileReaderWriterForMetadata(context.Background(),
				metadataFilePath, metadataConfig)

			if test.wantErr {
				assert.Nil(t, fileRW)
				assert.NotNil(t, err)
			} else {
				assert.NotNil(t, fileRW)
				assert.Nil(t, err)
			}

			err = os.RemoveAll(baseDir)
			assert.Nil(t, err)
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
						Text:        &notionapi.Text{Content: "Test Database"},
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

	filerw, err := rw.GetFileReaderWriter(context.Background(),
		TESTDATAPATH, true)
	assert.Nil(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			identifier, err := filerw.WriteDatabase(context.Background(),
				test.databaseObj)
			if test.wantErr {
				assert.Empty(t, identifier)
				assert.NotNil(t, err)
			} else {
				assert.NotEmpty(t, identifier)
				path := filepath.Join(TESTDATAPATH, rw.DATABASE_DIR_NAME,
					identifier.String())
				checkFilePermissions(t, path)
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
		name       string
		identifier string
		wantErr    bool
	}{
		{
			name:       "Valid database data from file",
			identifier: DATABASE_ID,
			wantErr:    false,
		},
		{
			name:       "Invalid json data",
			identifier: INVALID_JSON,
			wantErr:    true,
		},
		{
			name:       "Non Existing file",
			identifier: "xyz.json",
			wantErr:    true,
		},
	}

	filerw, err := rw.GetFileReaderWriter(context.Background(),
		RW_DATA_DIR, false)
	assert.Nil(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			database, err := filerw.ReadDatabase(context.Background(),
				rw.DataIdentifier(test.identifier))
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
								Text: &notionapi.Text{
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
								Text: &notionapi.Text{
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

	filerw, err := rw.GetFileReaderWriter(context.Background(),
		TESTDATAPATH, true)
	assert.Nil(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			identifier, err := filerw.WritePage(context.Background(), test.pageObj)
			if test.wantErr {
				assert.Empty(t, identifier)
				assert.NotNil(t, err)
			} else {
				assert.NotEmpty(t, identifier)
				path := filepath.Join(TESTDATAPATH, rw.PAGE_DIR_NAME,
					identifier.String())
				checkFilePermissions(t, path)
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
		name       string
		identifier string
		wantErr    bool
	}{
		{
			name:       "Valid page data from file",
			identifier: PAGE_ID,
			wantErr:    false,
		},
		{
			name:       "Invalid json data",
			identifier: INVALID_JSON,
			wantErr:    true,
		},
		{
			name:       "Non Existing file",
			identifier: "xyz.json",
			wantErr:    true,
		},
	}

	filerw, err := rw.GetFileReaderWriter(context.Background(),
		RW_DATA_DIR, false)
	assert.Nil(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			page, err := filerw.ReadPage(context.Background(),
				rw.DataIdentifier(test.identifier))
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

	filerw, err := rw.GetFileReaderWriter(context.Background(),
		TESTDATAPATH, true)
	assert.Nil(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			identifier, err := filerw.WriteBlock(context.Background(), test.blockObj)
			if test.wantErr {
				assert.Empty(t, identifier)
				assert.NotNil(t, err)
			} else {
				assert.NotEmpty(t, identifier)
				path := filepath.Join(TESTDATAPATH, rw.BLOCK_DIR_NAME,
					identifier.String())
				checkFilePermissions(t, path)
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
		name       string
		identifier string
		wantErr    bool
	}{
		{
			name:       "Valid block data from file",
			identifier: BLOCK_ID,
			wantErr:    false,
		},
		{
			name:       "Invalid json data",
			identifier: INVALID_JSON,
			wantErr:    true,
		},
		{
			name:       "Non Existing file",
			identifier: "xyz.json",
			wantErr:    true,
		},
	}

	filerw, err := rw.GetFileReaderWriter(context.Background(),
		RW_DATA_DIR, false)
	assert.Nil(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			block, err := filerw.ReadBlock(context.Background(),
				rw.DataIdentifier(test.identifier))
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

func TestCleanUp(t *testing.T) {
	database := &notionapi.Database{
		Object: notionapi.ObjectTypeDatabase,
		ID:     "some_id",
		Title: []notionapi.RichText{
			{
				Type:        notionapi.ObjectTypeText,
				Text:        &notionapi.Text{Content: "Test Database"},
				Annotations: &notionapi.Annotations{Color: "default"},
				PlainText:   "Test Database",
				Href:        "",
			},
		},
	}

	page := &notionapi.Page{
		Object: notionapi.ObjectTypePage,
		ID:     "some_id",
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
						Text: &notionapi.Text{
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
						Text: &notionapi.Text{
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
	}

	block := &notionapi.ChildPageBlock{
		BasicBlock: notionapi.BasicBlock{
			Object:      notionapi.ObjectTypeBlock,
			ID:          "some_id",
			Type:        notionapi.BlockTypeChildPage,
			HasChildren: true,
		},
		ChildPage: struct {
			Title string `json:"title"`
		}{
			Title: "Hello",
		},
	}

	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "CleanUp successful",
			wantErr: false,
		},
		{
			name:    "CleanUp failed",
			wantErr: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filerw, err := rw.GetFileReaderWriter(context.Background(),
				TESTDATAPATH, true)
			assert.Nil(t, err)
			id1, err := filerw.WriteBlock(context.Background(), block)
			assert.NotEmpty(t, id1)
			assert.Nil(t, err)
			id2, err := filerw.WriteDatabase(context.Background(), database)
			assert.NotEmpty(t, id2)
			assert.Nil(t, err)
			id3, err := filerw.WritePage(context.Background(), page)
			assert.NotEmpty(t, id3)
			assert.Nil(t, err)

			if test.wantErr {
				// Explicitly delete one file
				err := os.Remove(id2.String())
				assert.Nil(t, err)
				err = filerw.CleanUp(context.Background())
				assert.NotNil(t, err)
			} else {
				err = filerw.CleanUp(context.Background())
				assert.Nil(t, err)
			}
		})
	}
}

func TestWriteMetaData(t *testing.T) {
	t.Run("File write successful", func(t *testing.T) {
		filerw, err := rw.GetFileReaderWriter(context.Background(),
			TESTDATAPATH, true)
		assert.NotNil(t, filerw)
		assert.Nil(t, err)

		err = filerw.WriteMetaData(context.Background(), &metadata.MetaData{})
		assert.Nil(t, err)
	})

	// TODO: Add negative test cases
}

func TestFillStorageConfig(t *testing.T) {
	filerw, err := rw.GetFileReaderWriter(context.Background(),
		TESTDATAPATH, true)
	assert.NotNil(t, filerw)
	assert.Nil(t, err)

	expectedStorageConfig := &metadata.StorageConfig{
		Config: &metadata.StorageConfig_Local_{
			Local: &metadata.StorageConfig_Local{
				BlocksDir:   rw.BLOCK_DIR_NAME,
				PageDir:     rw.PAGE_DIR_NAME,
				DatabaseDir: rw.DATABASE_DIR_NAME,
			},
		},
	}

	storageConfig, err := filerw.GetStorageConfig(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, expectedStorageConfig, storageConfig)

}
