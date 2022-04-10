package rw_test

import (
	"os"
	"testing"

	"github.com/sawantshivaji1997/notionbackup/src/rw"
	"github.com/stretchr/testify/assert"
)

const (
	TESTDATAPATH      = "./../../testdata/"
	EXISTING_DIR_PATH = TESTDATAPATH
	NON_EXISTING_DIR  = TESTDATAPATH + "test_directory"
	NON_EXISTING_DIR2 = TESTDATAPATH + "test_directory2"
	INVALID_DIR_PATH  = "/xyz/sd/^7$%"
)

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
			name:                "Base directory does not exist",
			baseDirPath:         NON_EXISTING_DIR,
			createDirIfNotExist: true,
			wantErr:             false,
			cleanupRequied:      true,
		},
		{
			name:                "Base directory does not exist",
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
	}
}
