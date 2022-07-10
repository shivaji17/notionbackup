package builder_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/google/uuid"
	"github.com/jomei/notionapi"
	"github.com/sawantshivaji1997/notionbackup/src/mocks"
	"github.com/sawantshivaji1997/notionbackup/src/notionclient"
	"github.com/sawantshivaji1997/notionbackup/src/rw"
	"github.com/sawantshivaji1997/notionbackup/src/tree/builder"
	"github.com/sawantshivaji1997/notionbackup/src/tree/iterator"
	"github.com/sawantshivaji1997/notionbackup/src/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	OBJECT_TYPE_PAGE            = "Page"
	OBJECT_TYPE_DATABASE        = "Database"
	OBJECT_TYPE_BLOCK           = "Block"
	ERROR_STR                   = "error occurred"
	TEST_DATA_PATH              = "./../../../testdata/tree_builder/"
	WORKSPACE_TREE              = TEST_DATA_PATH + "workspace_tree.json"
	SPECIFIC_PAGE_DATABASE_TREE = TEST_DATA_PATH + "specific_page_database.json"
	EMPTY_CURSOR                = notionapi.Cursor("")
)

var errGeneric = fmt.Errorf(ERROR_STR)

func mockWritePage(m *mocks.ReaderWriter, param interface{}, err error) {
	m.On("WritePage", context.Background(), param).
		Return(rw.DataIdentifier(uuid.New().String()), err)
}

func mockWriteDatabase(m *mocks.ReaderWriter, param interface{}, err error) {
	m.On("WriteDatabase", context.Background(), param).
		Return(rw.DataIdentifier(uuid.New().String()), err)
}

func mockWriteBlock(m *mocks.ReaderWriter, param interface{}, err error) {
	m.On("WriteBlock", context.Background(), param).
		Return(rw.DataIdentifier(uuid.New().String()), err)
}

func insertIntoObjectIdMapping(objectMap map[string]map[string]bool,
	parentId string, childId string) {
	if valueMap, found := objectMap[parentId]; found {
		valueMap[childId] = true
		objectMap[parentId] = valueMap
		return
	}

	objectMap[parentId] = map[string]bool{
		childId: true,
	}
}

type notionTestData struct {
	WorkspaceTree bool
	ObjectList    []notionWrapperObject
}

type notionWrapperObject struct {
	ObjectType string
	Page       *notionapi.Page
	Database   *notionapi.Database
	Block      notionapi.Block
	ParentId   string
	ParentType string
}

// Helper struct used for creating mocks.NotionClient on-mock functions
type mocker struct {
	isWorkspaceTree     bool
	pageMap             map[string]*notionapi.Page
	databaseMap         map[string]*notionapi.Database
	pageId2BlockList    map[string][]notionapi.Block
	databaseId2PageList map[string][]notionapi.Page
	blockId2BlockList   map[string][]notionapi.Block
	mockedNotionClient  *mocks.NotionClient
	objectIdMapping     map[string]map[string]bool
	testData            *notionTestData
}

func getMocker(mockedNotionClient *mocks.NotionClient) *mocker {
	return &mocker{
		isWorkspaceTree: false,

		pageMap:     make(map[string]*notionapi.Page),
		databaseMap: make(map[string]*notionapi.Database),

		pageId2BlockList:    make(map[string][]notionapi.Block),
		databaseId2PageList: make(map[string][]notionapi.Page),
		blockId2BlockList:   make(map[string][]notionapi.Block),

		mockedNotionClient: mockedNotionClient,
		objectIdMapping:    make(map[string]map[string]bool),
		testData:           nil,
	}
}

func (c *mocker) mockGetPage(t *testing.T, pageId string) {
	for _, obj := range c.testData.ObjectList {
		if obj.ObjectType == OBJECT_TYPE_PAGE && obj.Page.ID.String() == pageId {
			page := obj.Page
			c.mockedNotionClient.On(
				"GetPageByID", context.Background(), notionclient.PageID(pageId)).
				Return(page, nil)
			return
		}
	}

	t.Fatalf("Page not found: %s", pageId)
}

// Reads the given file and loads the json data from file into notionTestData
// object
func (c *mocker) getNotionTestDataFromFile(t *testing.T,
	filePath string) {

	// Temporary structs as json from given file cannot be directly parsed into
	// notionTestData object
	type tempObject struct {
		ObjectType   string                 `json:"object_type"`
		Page         *notionapi.Page        `json:"page,omitempty"`
		Database     *notionapi.Database    `json:"database,omitempty"`
		TempBlockMap map[string]interface{} `json:"block,omitempty"`
		ParentId     string                 `json:"parent_id"`
		ParentType   string                 `json:"parent_type"`
	}
	type tempNotionTestData struct {
		WorkspaceTree bool         `json:"workspace_tree"`
		ObjectList    []tempObject `json:"object_list"`
	}

	jsonBytes, err := ioutil.ReadFile(filePath)
	assert.Nil(t, err)
	assert.NotEmpty(t, jsonBytes)

	testData := &tempNotionTestData{}
	err = json.Unmarshal(jsonBytes, &testData)
	assert.Nil(t, err)

	objList := []notionWrapperObject{}

	for _, obj := range testData.ObjectList {
		// if object type is block, it needs to decoded into specific block type
		if obj.ObjectType == OBJECT_TYPE_BLOCK {
			block, err := utils.DecodeBlockObject(obj.TempBlockMap)
			assert.Nil(t, err)
			newObj := &notionWrapperObject{
				ObjectType: obj.ObjectType,
				Page:       nil,
				Database:   nil,
				Block:      block,
				ParentId:   obj.ParentId,
				ParentType: obj.ParentType,
			}
			objList = append(objList, *newObj)
		} else {
			newObj := &notionWrapperObject{
				ObjectType: obj.ObjectType,
				Page:       obj.Page,
				Database:   obj.Database,
				Block:      nil,
				ParentId:   obj.ParentId,
				ParentType: obj.ParentType,
			}
			objList = append(objList, *newObj)
		}

	}

	c.testData = &notionTestData{
		WorkspaceTree: testData.WorkspaceTree,
		ObjectList:    objList,
	}
}

func (c *mocker) insertIntoDatabaseId2PageListMap(id string,
	object *notionapi.Page) {
	if id == "" {
		return
	}

	if !c.isWorkspaceTree {
		delete(c.pageMap, object.ID.String())
	}

	if dataList, found := c.databaseId2PageList[id]; found {
		c.databaseId2PageList[id] = append(dataList, *object)
		return
	}

	c.databaseId2PageList[id] = []notionapi.Page{*object}
}

func (c *mocker) insertIntoAnyId2BlockListMap(
	dataMap map[string][]notionapi.Block, id string, object notionapi.Block) {
	if id == "" {
		return
	}

	if dataList, found := dataMap[id]; found {
		dataMap[id] = append(dataList, object)
		return
	}

	dataMap[id] = []notionapi.Block{object}
}

// Creates a mapping of all possible relations for all types of objects
func (c *mocker) createMappings(t *testing.T, filePath string) {

	c.getNotionTestDataFromFile(t, filePath)
	c.isWorkspaceTree = c.testData.WorkspaceTree
	for _, obj := range c.testData.ObjectList {
		if obj.ObjectType == OBJECT_TYPE_PAGE {

			page := obj.Page
			insertIntoObjectIdMapping(
				c.objectIdMapping, obj.ParentId, page.ID.String())

			c.pageMap[page.ID.String()] = page

			if _, found := c.pageId2BlockList[page.ID.String()]; !found {
				c.pageId2BlockList[page.ID.String()] = []notionapi.Block{}
			}

			if obj.ParentId == "" {
				assert.Empty(t, obj.ParentType)
			} else if obj.ParentType == OBJECT_TYPE_DATABASE {
				c.insertIntoDatabaseId2PageListMap(obj.ParentId, page)
			} else if obj.ParentType == OBJECT_TYPE_BLOCK {
				assert.Equal(t, page.ID.String(), obj.ParentId)
			} else {
				t.Fatalf("Parent type %s not allowed for page", obj.ParentType)
			}

		} else if obj.ObjectType == OBJECT_TYPE_DATABASE {

			database := obj.Database
			insertIntoObjectIdMapping(
				c.objectIdMapping, obj.ParentId, database.ID.String())

			c.databaseMap[database.ID.String()] = database
			if _, found := c.databaseId2PageList[database.ID.String()]; !found {
				c.databaseId2PageList[database.ID.String()] = []notionapi.Page{}
			}

			if obj.ParentId == "" {
				assert.Empty(t, obj.ParentType)
			} else if obj.ParentType == OBJECT_TYPE_BLOCK {
				assert.Equal(t, database.ID.String(), obj.ParentId)
			} else {
				t.Fatalf("Parent type %s not allowed for database", obj.ParentType)
			}

		} else if obj.ObjectType == OBJECT_TYPE_BLOCK {

			block := obj.Block
			// Blocks objects must have a parent
			assert.NotEmpty(t, obj.ParentId)
			assert.NotEmpty(t, obj.ParentType)
			insertIntoObjectIdMapping(
				c.objectIdMapping, obj.ParentId, block.GetID().String())

			if obj.ParentType == OBJECT_TYPE_PAGE {
				c.insertIntoAnyId2BlockListMap(c.pageId2BlockList, obj.ParentId, block)
			} else if obj.ParentType == OBJECT_TYPE_BLOCK {
				c.insertIntoAnyId2BlockListMap(c.blockId2BlockList, obj.ParentId, block)
			} else {
				t.Fatalf("Parent type %s not allowed for block", obj.ParentType)
			}

		} else {
			t.Fatalf("Unknown object Type: %s", obj.ObjectType)
		}
	}
}

// This function will mock functions from mocks.NotionClient with required
// parameters and return types
func (c *mocker) mockNotionClientFunctions() {
	cursor := notionapi.Cursor(uuid.NewString())
	if c.isWorkspaceTree {
		pageList := []notionapi.Page{}
		databaseList := []notionapi.Database{}

		for _, page := range c.pageMap {
			temp := page
			pageList = append(pageList, *temp)
		}

		for _, database := range c.databaseMap {
			temp := database
			databaseList = append(databaseList, *temp)
		}

		index := len(pageList) / 2
		c.mockedNotionClient.On("GetAllPages", context.Background(), EMPTY_CURSOR).
			Return(pageList[:index], cursor, nil)
		c.mockedNotionClient.On("GetAllPages", context.Background(), cursor).
			Return(pageList[index:], EMPTY_CURSOR, nil)

		index = len(databaseList) / 2
		c.mockedNotionClient.On(
			"GetAllDatabases", context.Background(), EMPTY_CURSOR).
			Return(databaseList[:index], cursor, nil)

		c.mockedNotionClient.On("GetAllDatabases", context.Background(), cursor).
			Return(databaseList[index:], EMPTY_CURSOR, nil)

		for databaseId, pages := range c.databaseId2PageList {
			if len(pages) == 0 {
				c.mockedNotionClient.On("GetDatabasePages", context.Background(),
					notionclient.DatabaseID(databaseId), EMPTY_CURSOR).
					Return(pages, EMPTY_CURSOR, nil)
			}
		}

	} else {

		for pageId, page := range c.pageMap {
			c.mockedNotionClient.On(
				"GetPageByID", context.Background(), notionclient.PageID(pageId)).
				Return(page, nil)
		}

		for databaseId, database := range c.databaseMap {
			c.mockedNotionClient.On("GetDatabaseByID", context.Background(),
				notionclient.DatabaseID(databaseId)).Return(database, nil)
		}

		for databaseId, pages := range c.databaseId2PageList {
			if len(pages) <= 1 {
				c.mockedNotionClient.On("GetDatabasePages", context.Background(),
					notionclient.DatabaseID(databaseId), EMPTY_CURSOR).
					Return(pages, EMPTY_CURSOR, nil)
				continue
			}

			index := len(pages) / 2
			c.mockedNotionClient.On("GetDatabasePages", context.Background(),
				notionclient.DatabaseID(databaseId), EMPTY_CURSOR).
				Return(pages[:index], cursor, nil)

			c.mockedNotionClient.On("GetDatabasePages", context.Background(),
				notionclient.DatabaseID(databaseId), cursor).
				Return(pages[index:], EMPTY_CURSOR, nil)
		}

	}

	for pageId, blocks := range c.pageId2BlockList {
		if len(blocks) <= 1 {
			c.mockedNotionClient.On("GetPageBlocks", context.Background(),
				notionclient.PageID(pageId), EMPTY_CURSOR).
				Return(blocks, EMPTY_CURSOR, nil)

			continue
		}

		index := len(blocks) / 2
		c.mockedNotionClient.On("GetPageBlocks", context.Background(),
			notionclient.PageID(pageId), EMPTY_CURSOR).
			Return(blocks[:index], cursor, nil)

		c.mockedNotionClient.On("GetPageBlocks", context.Background(),
			notionclient.PageID(pageId), cursor).
			Return(blocks[index:], EMPTY_CURSOR, nil)
	}

	for blockId, blocks := range c.blockId2BlockList {
		if len(blocks) <= 1 {
			c.mockedNotionClient.On("GetChildBlocksOfBlock", context.Background(),
				notionclient.BlockID(blockId), EMPTY_CURSOR).
				Return(blocks, EMPTY_CURSOR, nil)
			continue
		}

		index := len(blocks) / 2
		c.mockedNotionClient.On("GetChildBlocksOfBlock", context.Background(),
			notionclient.BlockID(blockId), EMPTY_CURSOR).
			Return(blocks[:index], cursor, nil)

		c.mockedNotionClient.On("GetChildBlocksOfBlock", context.Background(),
			notionclient.BlockID(blockId), cursor).
			Return(blocks[index:], EMPTY_CURSOR, nil)
	}

}

// ExportTreebuilder tester
func TestExportTreeBuilder(t *testing.T) {
	assert := assert.New(t)

	//////////////////////////////////////////////////////////////////////////////
	t.Run("Error while fetching all pages", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		mockedRW.On("CleanUp", context.Background()).Return(nil)
		// mock all required NotionClient functions
		mockedNotionClient.On(
			"GetAllPages", context.Background(), notionapi.Cursor("")).
			Return(
				make([]notionapi.Page, 0), notionapi.Cursor(""), errGeneric)

		treeBuilder := builder.GetExportTreebuilder(context.Background(),
			mockedNotionClient, mockedRW, &builder.TreeBuilderRequest{})
		tree, err := treeBuilder.BuildTree(context.Background())
		assert.NotNil(err)
		assert.Nil(tree)
	})

	//////////////////////////////////////////////////////////////////////////////
	t.Run("Error while fetching all databases", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		mockedRW.On("CleanUp", context.Background()).Return(nil)

		// mock all required NotionClient functions
		mockedNotionClient.On(
			"GetAllPages", context.Background(), notionapi.Cursor("")).
			Return(make([]notionapi.Page, 0), notionapi.Cursor(""), nil)

		mockedNotionClient.On(
			"GetAllDatabases", context.Background(), notionapi.Cursor("")).
			Return(make(
				[]notionapi.Database, 0), notionapi.Cursor(""), errGeneric)

		treeBuilder := builder.GetExportTreebuilder(context.Background(),
			mockedNotionClient, mockedRW, &builder.TreeBuilderRequest{})

		tree, err := treeBuilder.BuildTree(context.Background())
		assert.NotNil(err)
		assert.Nil(tree)
	})

	//////////////////////////////////////////////////////////////////////////////
	t.Run("Error while writing all pages", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		// mock ReaderWriter functions
		mockWritePage(mockedRW, mock.Anything, errGeneric)
		mockedRW.On("CleanUp", context.Background()).Return(nil)

		// mock all required NotionClient functions
		page := &notionapi.Page{Parent: notionapi.Parent{
			Type: notionapi.ParentType("workspace"),
		}}
		mockedNotionClient.On(
			"GetAllPages", context.Background(), notionapi.Cursor("")).
			Return([]notionapi.Page{*page}, notionapi.Cursor(""), nil)

		treeBuilder := builder.GetExportTreebuilder(context.Background(),
			mockedNotionClient, mockedRW, &builder.TreeBuilderRequest{})

		tree, err := treeBuilder.BuildTree(context.Background())
		assert.NotNil(err)
		assert.Nil(tree)
	})

	//////////////////////////////////////////////////////////////////////////////
	t.Run("Error while writing all databases", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		// mock ReaderWirter functions
		mockWriteDatabase(mockedRW, mock.Anything, errGeneric)
		mockedRW.On("CleanUp", context.Background()).Return(nil)

		// mock all required NotionClient functions
		mockedNotionClient.On(
			"GetAllPages", context.Background(), notionapi.Cursor("")).
			Return(make([]notionapi.Page, 0), notionapi.Cursor(""), nil)

		database := &notionapi.Database{
			Parent: notionapi.Parent{
				Type: notionapi.ParentType("workspace"),
			},
		}
		mockedNotionClient.On(
			"GetAllDatabases", context.Background(), notionapi.Cursor("")).
			Return([]notionapi.Database{*database}, notionapi.Cursor(""), nil)

		treeBuilder := builder.GetExportTreebuilder(context.Background(),
			mockedNotionClient, mockedRW, &builder.TreeBuilderRequest{})
		tree, err := treeBuilder.BuildTree(context.Background())
		assert.NotNil(err)
		assert.Nil(tree)
	})

	//////////////////////////////////////////////////////////////////////////////
	t.Run("Build tree for whole workspace", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		// mock all ReaderWriter functions
		mockWritePage(mockedRW, mock.Anything, nil)
		mockWriteDatabase(mockedRW, mock.Anything, nil)
		mockWriteBlock(mockedRW, mock.Anything, nil)

		// mock all required NotionClient functions
		mockerObj := getMocker(mockedNotionClient)
		mockerObj.createMappings(t, WORKSPACE_TREE)
		mockerObj.mockNotionClientFunctions()
		treeBuilder := builder.GetExportTreebuilder(context.Background(),
			mockedNotionClient, mockedRW, &builder.TreeBuilderRequest{})
		tree, err := treeBuilder.BuildTree(context.Background())
		assert.Nil(err)
		assert.NotNil(tree)

		actualObjectMapping := make(map[string]map[string]bool, 0)
		childIter := iterator.GetChildIterator(tree.RootNode)
		for {
			obj, err := childIter.Next()
			if err == iterator.ErrDone {
				break
			}
			insertIntoObjectIdMapping(actualObjectMapping,
				tree.RootNode.GetNotionObjectId(), obj.GetNotionObjectId())
		}

		treeIter := iterator.GetTreeIterator(tree.RootNode)
		for {
			obj, err := treeIter.Next()
			if err == iterator.ErrDone {
				break
			}

			childIter := iterator.GetChildIterator(obj)
			for {
				childObj, err := childIter.Next()
				if err == iterator.ErrDone {
					break
				}
				insertIntoObjectIdMapping(actualObjectMapping, obj.GetNotionObjectId(),
					childObj.GetNotionObjectId())
			}
		}

		assert.Equal(mockerObj.objectIdMapping, actualObjectMapping)
	})

	//////////////////////////////////////////////////////////////////////////////
	t.Run("Error while fetching given page", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		mockedRW.On("CleanUp", context.Background()).Return(nil)

		// mock all required NotionClient functions
		mockedNotionClient.On("GetPageByID", context.Background(),
			notionclient.PageID("36dac6ee-76e9-4c99-94a9-b0989be3f624")).
			Return(nil, errGeneric)

		treeBuilder := builder.GetExportTreebuilder(context.Background(),
			mockedNotionClient, mockedRW,
			&builder.TreeBuilderRequest{
				PageIdList: []string{"36dac6ee-76e9-4c99-94a9-b0989be3f624"},
			})
		tree, err := treeBuilder.BuildTree(context.Background())
		assert.NotNil(err)
		assert.Nil(tree)
	})

	//////////////////////////////////////////////////////////////////////////////
	t.Run("Error while writing page for given page", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		// mock ReaderWriter functions
		mockWritePage(mockedRW, mock.Anything, errGeneric)
		mockedRW.On("CleanUp", context.Background()).Return(nil)

		// mock all required NotionClient functions
		mockedNotionClient.On("GetPageByID", context.Background(),
			notionclient.PageID("36dac6ee-76e9-4c99-94a9-b0989be3f624")).
			Return(&notionapi.Page{}, nil)

		treeBuilder := builder.GetExportTreebuilder(context.Background(),
			mockedNotionClient, mockedRW,
			&builder.TreeBuilderRequest{
				PageIdList: []string{"36dac6ee-76e9-4c99-94a9-b0989be3f624"},
			})

		tree, err := treeBuilder.BuildTree(context.Background())
		assert.NotNil(err)
		assert.Nil(tree)
	})

	//////////////////////////////////////////////////////////////////////////////
	t.Run("Error while fetching page blocks for given page", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		// Mock ReaderWriter functions
		mockWritePage(mockedRW, mock.Anything, nil)
		mockedRW.On("CleanUp", context.Background()).Return(nil)

		// mock all required NotionClient functions
		mockedNotionClient.On("GetPageByID", context.Background(),
			notionclient.PageID("36dac6ee-76e9-4c99-94a9-b0989be3f624")).
			Return(&notionapi.Page{
				ID: notionapi.ObjectID("36dac6ee-76e9-4c99-94a9-b0989be3f624"),
			}, nil)

		mockedNotionClient.On(
			"GetPageBlocks", context.Background(),
			notionclient.PageID("36dac6ee-76e9-4c99-94a9-b0989be3f624"),
			EMPTY_CURSOR).
			Return([]notionapi.Block{}, EMPTY_CURSOR, errGeneric)

		treeBuilder := builder.GetExportTreebuilder(context.Background(),
			mockedNotionClient, mockedRW,
			&builder.TreeBuilderRequest{
				PageIdList: []string{"36dac6ee-76e9-4c99-94a9-b0989be3f624"},
			})

		tree, err := treeBuilder.BuildTree(context.Background())
		assert.NotNil(err)
		assert.Nil(tree)
	})

	//////////////////////////////////////////////////////////////////////////////
	t.Run("Error while fetching given database", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		mockedRW.On("CleanUp", context.Background()).Return(nil)

		// mock all required NotionClient functions
		mockedNotionClient.On(
			"GetDatabaseByID", context.Background(),
			notionclient.DatabaseID("36dac6ee-76e9-4c99-94a9-b0989be3f624")).
			Return(nil, errGeneric)

		treeBuilder := builder.GetExportTreebuilder(
			context.Background(), mockedNotionClient, mockedRW,
			&builder.TreeBuilderRequest{
				DatabaseIdList: []string{"36dac6ee-76e9-4c99-94a9-b0989be3f624"},
			})

		tree, err := treeBuilder.BuildTree(context.Background())
		assert.NotNil(err)
		assert.Nil(tree)
	})

	//////////////////////////////////////////////////////////////////////////////
	t.Run("Error while writing database with given database", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		// mock ReaderWriter functions
		mockWriteDatabase(mockedRW, mock.Anything, errGeneric)
		mockedRW.On("CleanUp", context.Background()).Return(nil)

		// mock all required NotionClient functions
		mockedNotionClient.On(
			"GetDatabaseByID", context.Background(),
			notionclient.DatabaseID(
				"36dac6ee-76e9-4c99-94a9-b0989be3f624")).
			Return(&notionapi.Database{}, nil)

		treeBuilder := builder.GetExportTreebuilder(
			context.Background(), mockedNotionClient, mockedRW,
			&builder.TreeBuilderRequest{
				DatabaseIdList: []string{"36dac6ee-76e9-4c99-94a9-b0989be3f624"},
			})
		tree, err := treeBuilder.BuildTree(context.Background())
		assert.NotNil(err)
		assert.Nil(tree)
	})

	//////////////////////////////////////////////////////////////////////////////
	t.Run("Error while fetching page for given database", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		// Mock ReaderWriter functions
		mockWriteDatabase(mockedRW, mock.Anything, nil)
		mockedRW.On("CleanUp", context.Background()).Return(nil)

		// mock all required NotionClient functions
		mockedNotionClient.On("GetDatabaseByID", context.Background(),
			notionclient.DatabaseID("36dac6ee-76e9-4c99-94a9-b0989be3f624")).
			Return(&notionapi.Database{
				ID: notionapi.ObjectID("36dac6ee-76e9-4c99-94a9-b0989be3f624"),
			}, nil)

		mockedNotionClient.On(
			"GetDatabasePages", context.Background(),
			notionclient.DatabaseID("36dac6ee-76e9-4c99-94a9-b0989be3f624"),
			EMPTY_CURSOR).
			Return([]notionapi.Page{}, EMPTY_CURSOR, errGeneric)

		treeBuilder := builder.GetExportTreebuilder(context.Background(),
			mockedNotionClient, mockedRW,
			&builder.TreeBuilderRequest{
				DatabaseIdList: []string{"36dac6ee-76e9-4c99-94a9-b0989be3f624"},
			})

		tree, err := treeBuilder.BuildTree(context.Background())
		assert.NotNil(err)
		assert.Nil(tree)
	})

	//////////////////////////////////////////////////////////////////////////////
	t.Run("Error while writing block", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		// Mock ReaderWriter functions
		mockWritePage(mockedRW, mock.Anything, nil)
		mockWriteBlock(mockedRW, mock.Anything, errGeneric)
		mockedRW.On("CleanUp", context.Background()).Return(nil)

		// mock all required NotionClient functions
		mockedNotionClient.On("GetPageByID", context.Background(),
			notionclient.PageID("36dac6ee-76e9-4c99-94a9-b0989be3f624")).
			Return(&notionapi.Page{
				ID: notionapi.ObjectID("36dac6ee-76e9-4c99-94a9-b0989be3f624"),
			}, nil)

		blockId := uuid.NewString()
		block := &notionapi.ColumnBlock{
			BasicBlock: notionapi.BasicBlock{
				ID:          notionapi.BlockID(blockId),
				HasChildren: true,
			},
		}

		mockedNotionClient.On(
			"GetPageBlocks", context.Background(),
			notionclient.PageID("36dac6ee-76e9-4c99-94a9-b0989be3f624"),
			EMPTY_CURSOR).
			Return([]notionapi.Block{block}, EMPTY_CURSOR, nil)

		treeBuilder := builder.GetExportTreebuilder(context.Background(),
			mockedNotionClient, mockedRW,
			&builder.TreeBuilderRequest{
				PageIdList: []string{"36dac6ee-76e9-4c99-94a9-b0989be3f624"},
			})

		tree, err := treeBuilder.BuildTree(context.Background())
		assert.NotNil(err)
		assert.Nil(tree)
	})

	//////////////////////////////////////////////////////////////////////////////
	t.Run("Error while fetching blocks of given blocks", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		// Mock ReaderWriter functions
		mockWriteBlock(mockedRW, mock.Anything, nil)
		mockWritePage(mockedRW, mock.Anything, nil)
		mockedRW.On("CleanUp", context.Background()).Return(nil)

		// mock all required NotionClient functions
		mockedNotionClient.On("GetPageByID", context.Background(),
			notionclient.PageID("36dac6ee-76e9-4c99-94a9-b0989be3f624")).
			Return(&notionapi.Page{
				ID: notionapi.ObjectID("36dac6ee-76e9-4c99-94a9-b0989be3f624"),
			}, nil)

		blockId := uuid.NewString()
		block := &notionapi.ColumnBlock{
			BasicBlock: notionapi.BasicBlock{
				ID:          notionapi.BlockID(blockId),
				HasChildren: true,
			},
		}

		mockedNotionClient.On(
			"GetPageBlocks", context.Background(),
			notionclient.PageID("36dac6ee-76e9-4c99-94a9-b0989be3f624"),
			EMPTY_CURSOR).
			Return([]notionapi.Block{block}, EMPTY_CURSOR, nil)

		mockedNotionClient.On(
			"GetChildBlocksOfBlock", context.Background(),
			notionclient.BlockID(blockId), EMPTY_CURSOR).
			Return([]notionapi.Block{}, EMPTY_CURSOR, errGeneric)

		treeBuilder := builder.GetExportTreebuilder(context.Background(),
			mockedNotionClient, mockedRW,
			&builder.TreeBuilderRequest{
				PageIdList: []string{"36dac6ee-76e9-4c99-94a9-b0989be3f624"},
			})

		tree, err := treeBuilder.BuildTree(context.Background())
		assert.NotNil(err)
		assert.Nil(tree)
	})

	//////////////////////////////////////////////////////////////////////////////
	t.Run("Build tree for given page and database", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		// mock all ReaderWriter functions
		mockWritePage(mockedRW, mock.Anything, nil)
		mockWriteDatabase(mockedRW, mock.Anything, nil)
		mockWriteBlock(mockedRW, mock.Anything, nil)

		// mock all required NotionClient functions
		mockerObj := getMocker(mockedNotionClient)
		mockerObj.createMappings(t, SPECIFIC_PAGE_DATABASE_TREE)
		mockerObj.mockNotionClientFunctions()
		treeBuilder := builder.GetExportTreebuilder(
			context.Background(), mockedNotionClient, mockedRW,
			&builder.TreeBuilderRequest{
				PageIdList:     []string{"05034203-2870-4bc8-b1f9-22c0ae6e56ba"},
				DatabaseIdList: []string{"5ed2d97a-510a-4756-b113-cc28c7a30fd7"},
			})

		tree, err := treeBuilder.BuildTree(context.Background())
		assert.Nil(err)
		assert.NotNil(tree)
		tree2, err := treeBuilder.BuildTree(context.Background())
		assert.Nil(err)
		assert.NotNil(tree2)
		assert.Equal(tree, tree2)

		actualObjectMapping := make(map[string]map[string]bool, 0)
		childIter := iterator.GetChildIterator(tree.RootNode)
		for {
			obj, err := childIter.Next()
			if err == iterator.ErrDone {
				break
			}
			insertIntoObjectIdMapping(actualObjectMapping,
				tree.RootNode.GetNotionObjectId(), obj.GetNotionObjectId())
		}

		treeIter := iterator.GetTreeIterator(tree.RootNode)
		for {
			obj, err := treeIter.Next()
			if err == iterator.ErrDone {
				break
			}

			childIter := iterator.GetChildIterator(obj)
			for {
				childObj, err := childIter.Next()
				if err == iterator.ErrDone {
					break
				}
				insertIntoObjectIdMapping(actualObjectMapping,
					obj.GetNotionObjectId(), childObj.GetNotionObjectId())
			}
		}

		assert.Equal(mockerObj.objectIdMapping, actualObjectMapping)
	})

	//////////////////////////////////////////////////////////////////////////////
	t.Run("Build tree for given pages and databases where some page/database "+
		"is child other page and database", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		// mock all ReaderWriter functions
		mockWritePage(mockedRW, mock.Anything, nil)
		mockWriteDatabase(mockedRW, mock.Anything, nil)
		mockWriteBlock(mockedRW, mock.Anything, nil)

		// mock all required NotionClient functions
		mockerObj := getMocker(mockedNotionClient)
		mockerObj.createMappings(t, SPECIFIC_PAGE_DATABASE_TREE)

		mockerObj.mockGetPage(t, "bd51fa91-079a-40c6-98d1-658060d62e39")
		mockerObj.mockGetPage(t, "22780993-87f6-43fe-bc12-e9223b15e303")

		mockerObj.mockNotionClientFunctions()
		treeBuilder := builder.GetExportTreebuilder(
			context.Background(), mockedNotionClient, mockedRW,
			&builder.TreeBuilderRequest{
				PageIdList: []string{"05034203-2870-4bc8-b1f9-22c0ae6e56ba",
					"53d18605-7779-4700-b16d-662a332283a1",
					"bd51fa91-079a-40c6-98d1-658060d62e39",
					"22780993-87f6-43fe-bc12-e9223b15e303"},
				DatabaseIdList: []string{"5ed2d97a-510a-4756-b113-cc28c7a30fd7",
					"9cd00ee9-63e5-4dad-b0aa-d76f2ecc36d1"},
			})

		tree, err := treeBuilder.BuildTree(context.Background())
		assert.Nil(err)
		assert.NotNil(tree)

		actualObjectMapping := make(map[string]map[string]bool, 0)
		childIter := iterator.GetChildIterator(tree.RootNode)
		for {
			obj, err := childIter.Next()
			if err == iterator.ErrDone {
				break
			}
			insertIntoObjectIdMapping(actualObjectMapping,
				tree.RootNode.GetNotionObjectId(), obj.GetNotionObjectId())
		}

		treeIter := iterator.GetTreeIterator(tree.RootNode)
		for {
			obj, err := treeIter.Next()
			if err == iterator.ErrDone {
				break
			}

			childIter := iterator.GetChildIterator(obj)
			for {
				childObj, err := childIter.Next()
				if err == iterator.ErrDone {
					break
				}
				insertIntoObjectIdMapping(actualObjectMapping,
					obj.GetNotionObjectId(), childObj.GetNotionObjectId())
			}
		}

		assert.Equal(mockerObj.objectIdMapping, actualObjectMapping)
	})
}
