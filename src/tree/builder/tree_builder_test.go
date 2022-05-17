package builder_test

import (
	"context"
	"encoding/json"
	"errors"
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

func insertIntoObjectIdMapping(objectMap map[string]map[string]bool, keyId string, valueId string) {
	if valueMap, found := objectMap[keyId]; found {
		valueMap[valueId] = true
		objectMap[keyId] = valueMap
		return
	}

	objectMap[keyId] = map[string]bool{
		valueId: true,
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

type mocker struct {
	isWorkspaceTree     bool
	pageMap             map[string]*notionapi.Page
	databaseMap         map[string]*notionapi.Database
	pageId2BlockList    map[string][]notionapi.Block
	databaseId2PageList map[string][]notionapi.Page
	blockId2BlockList   map[string][]notionapi.Block
	mockedNotionClient  *mocks.NotionClient
	objectIdMapping     map[string]map[string]bool
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
	}
}

func (c *mocker) getNotionTestDataFromFile(t *testing.T, filePath string) *notionTestData {

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

	return &notionTestData{
		WorkspaceTree: testData.WorkspaceTree,
		ObjectList:    objList,
	}
}

func (c *mocker) insertIntoDatabaseId2PageListMap(id string, object *notionapi.Page) {
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

func (c *mocker) insertIntoAnyId2BlockListMap(dataMap map[string][]notionapi.Block, id string, object notionapi.Block) {
	if id == "" {
		return
	}

	if dataList, found := dataMap[id]; found {
		dataMap[id] = append(dataList, object)
		return
	}

	dataMap[id] = []notionapi.Block{object}
}

// This function will mock functions from mocks.NotionClient with required
// parameters and return types
func (c *mocker) createMappings(t *testing.T, filePath string) {

	testData := c.getNotionTestDataFromFile(t, filePath)
	c.isWorkspaceTree = testData.WorkspaceTree
	for _, obj := range testData.ObjectList {
		if obj.ObjectType == OBJECT_TYPE_PAGE {

			page := obj.Page
			insertIntoObjectIdMapping(c.objectIdMapping, obj.ParentId, page.ID.String())

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
				t.Fatal("Parent type " + obj.ParentType + " not allowed for page")
			}

		} else if obj.ObjectType == OBJECT_TYPE_DATABASE {

			database := obj.Database
			insertIntoObjectIdMapping(c.objectIdMapping, obj.ParentId, database.ID.String())

			c.databaseMap[database.ID.String()] = database
			if _, found := c.databaseId2PageList[database.ID.String()]; !found {
				c.databaseId2PageList[database.ID.String()] = []notionapi.Page{}
			}

			if obj.ParentId == "" {
				assert.Empty(t, obj.ParentType)
			} else if obj.ParentType == OBJECT_TYPE_BLOCK {
				assert.Equal(t, database.ID.String(), obj.ParentId)
			} else {
				t.Fatal("Parent type " + obj.ParentType + " not allowed for database")
			}

		} else if obj.ObjectType == OBJECT_TYPE_BLOCK {

			block := obj.Block
			// Blocks objects must have a parent
			assert.NotEmpty(t, obj.ParentId)
			assert.NotEmpty(t, obj.ParentType)
			insertIntoObjectIdMapping(c.objectIdMapping, obj.ParentId, block.GetID().String())

			if obj.ParentType == OBJECT_TYPE_PAGE {
				c.insertIntoAnyId2BlockListMap(c.pageId2BlockList, obj.ParentId, block)
			} else if obj.ParentType == OBJECT_TYPE_BLOCK {
				c.insertIntoAnyId2BlockListMap(c.blockId2BlockList, obj.ParentId, block)
			} else {
				t.Fatal("Parent type " + obj.ParentType + " not allowed for block")
			}

		} else {
			t.Fatal("Unknown object Type: " + obj.ObjectType)
		}
	}
}

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
		c.mockedNotionClient.On("GetAllDatabases", context.Background(), EMPTY_CURSOR).
			Return(databaseList[:index], cursor, nil)
		c.mockedNotionClient.On("GetAllDatabases", context.Background(), cursor).
			Return(databaseList[index:], EMPTY_CURSOR, nil)

		for databaseId, pages := range c.databaseId2PageList {
			if len(pages) == 0 {
				c.mockedNotionClient.On("GetDatabasePages", context.Background(), notionclient.DatabaseID(databaseId), EMPTY_CURSOR).
					Return(pages, EMPTY_CURSOR, nil)
			}
		}

	} else {

		for pageId, page := range c.pageMap {
			c.mockedNotionClient.On("GetPageByID", context.Background(), notionclient.PageID(pageId)).
				Return(page, nil)
		}

		for databaseId, database := range c.databaseMap {
			c.mockedNotionClient.On("GetDatabaseByID", context.Background(), notionclient.DatabaseID(databaseId)).
				Return(database, nil)
		}

		for databaseId, pages := range c.databaseId2PageList {
			if len(pages) <= 1 {
				c.mockedNotionClient.On("GetDatabasePages", context.Background(), notionclient.DatabaseID(databaseId), EMPTY_CURSOR).
					Return(pages, EMPTY_CURSOR, nil)
				continue
			}

			index := len(pages) / 2
			c.mockedNotionClient.On("GetDatabasePages", context.Background(), notionclient.DatabaseID(databaseId), EMPTY_CURSOR).
				Return(pages[:index], cursor, nil)
			c.mockedNotionClient.On("GetDatabasePages", context.Background(), notionclient.DatabaseID(databaseId), cursor).
				Return(pages[index:], EMPTY_CURSOR, nil)
		}

	}

	for pageId, blocks := range c.pageId2BlockList {
		if len(blocks) <= 1 {
			c.mockedNotionClient.On("GetPageBlocks", context.Background(), notionclient.PageID(pageId), EMPTY_CURSOR).
				Return(blocks, EMPTY_CURSOR, nil)

			continue
		}

		index := len(blocks) / 2
		c.mockedNotionClient.On("GetPageBlocks", context.Background(), notionclient.PageID(pageId), EMPTY_CURSOR).
			Return(blocks[:index], cursor, nil)
		c.mockedNotionClient.On("GetPageBlocks", context.Background(), notionclient.PageID(pageId), cursor).
			Return(blocks[index:], EMPTY_CURSOR, nil)
	}

	for blockId, blocks := range c.blockId2BlockList {
		if len(blocks) <= 1 {
			c.mockedNotionClient.On("GetChildBlocksOfBlock", context.Background(), notionclient.BlockID(blockId), EMPTY_CURSOR).
				Return(blocks, EMPTY_CURSOR, nil)
			continue
		}

		index := len(blocks) / 2
		c.mockedNotionClient.On("GetChildBlocksOfBlock", context.Background(), notionclient.BlockID(blockId), EMPTY_CURSOR).
			Return(blocks[:index], cursor, nil)
		c.mockedNotionClient.On("GetChildBlocksOfBlock", context.Background(), notionclient.BlockID(blockId), cursor).
			Return(blocks[index:], EMPTY_CURSOR, nil)
	}

}

func TestExportTreeBuilder(t *testing.T) {
	assert := assert.New(t)

	t.Run("Get root node without building tree", func(t *testing.T) {
		treeBuilder := builder.GetExportTreebuilder(context.Background(), mocks.NewNotionClient(t), mocks.NewReaderWriter(t), &builder.TreeRequest{})
		rootNode, err := treeBuilder.GetRootNode()
		assert.Nil(rootNode)
		assert.NotNil(err)
	})

	t.Run("Error while fetching all pages", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		// mock all required NotionClient functions
		mockedNotionClient.On("GetAllPages", context.Background(), notionapi.Cursor("")).Return(make([]notionapi.Page, 0), notionapi.Cursor(""), errors.New(ERROR_STR))

		treeBuilder := builder.GetExportTreebuilder(context.Background(), mockedNotionClient, mockedRW, &builder.TreeRequest{})
		err := treeBuilder.BuildTree(context.Background())
		assert.NotNil(err)
	})

	t.Run("Error while fetching all databases", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		// mock all required NotionClient functions
		mockedNotionClient.On("GetAllPages", context.Background(), notionapi.Cursor("")).Return(make([]notionapi.Page, 0), notionapi.Cursor(""), nil)
		mockedNotionClient.On("GetAllDatabases", context.Background(), notionapi.Cursor("")).
			Return(make([]notionapi.Database, 0), notionapi.Cursor(""), errors.New(ERROR_STR))

		treeBuilder := builder.GetExportTreebuilder(context.Background(), mockedNotionClient, mockedRW, &builder.TreeRequest{})
		err := treeBuilder.BuildTree(context.Background())
		assert.NotNil(err)
	})

	t.Run("Error while writing all pages", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		mockedRW.On("WritePage", context.Background(), mock.Anything).Return(rw.DataIdentifier(""), errors.New(ERROR_STR))
		// mock all required NotionClient functions
		page := &notionapi.Page{Parent: notionapi.Parent{
			Type: notionapi.ParentType("workspace"),
		}}
		mockedNotionClient.On("GetAllPages", context.Background(), notionapi.Cursor("")).Return([]notionapi.Page{*page}, notionapi.Cursor(""), nil)

		treeBuilder := builder.GetExportTreebuilder(context.Background(), mockedNotionClient, mockedRW, &builder.TreeRequest{})
		err := treeBuilder.BuildTree(context.Background())
		assert.NotNil(err)
	})

	t.Run("Error while writing all databases", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		database := &notionapi.Database{
			Parent: notionapi.Parent{
				Type: notionapi.ParentType("workspace"),
			},
		}
		// mock all required NotionClient functions
		mockedNotionClient.On("GetAllPages", context.Background(), notionapi.Cursor("")).Return(make([]notionapi.Page, 0), notionapi.Cursor(""), nil)
		mockedNotionClient.On("GetAllDatabases", context.Background(), notionapi.Cursor("")).
			Return([]notionapi.Database{*database}, notionapi.Cursor(""), nil)
		mockedRW.On("WriteDatabase", context.Background(), mock.Anything).Return(rw.DataIdentifier(""), errors.New(ERROR_STR))

		treeBuilder := builder.GetExportTreebuilder(context.Background(), mockedNotionClient, mockedRW, &builder.TreeRequest{})
		err := treeBuilder.BuildTree(context.Background())
		assert.NotNil(err)
	})

	t.Run("Build tree for whole workspace", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		// mock all ReaderWriter functions
		mockedRW.On("WriteDatabase", context.Background(), mock.Anything).Return(rw.DataIdentifier(uuid.New().String()), nil)
		mockedRW.On("WritePage", context.Background(), mock.Anything).Return(rw.DataIdentifier(uuid.New().String()), nil)
		mockedRW.On("WriteBlock", context.Background(), mock.Anything).Return(rw.DataIdentifier(uuid.New().String()), nil)

		// mock all required NotionClient functions
		mockerObj := getMocker(mockedNotionClient)
		mockerObj.createMappings(t, WORKSPACE_TREE)
		mockerObj.mockNotionClientFunctions()
		treeBuilder := builder.GetExportTreebuilder(context.Background(), mockedNotionClient, mockedRW, &builder.TreeRequest{})
		err := treeBuilder.BuildTree(context.Background())
		assert.Nil(err)
		rootNode, err := treeBuilder.GetRootNode()
		assert.NotNil(rootNode)
		assert.Nil(err)

		actualObjectMapping := make(map[string]map[string]bool, 0)
		childIter := iterator.GetChildIterator(rootNode)
		for {
			obj, err := childIter.Next()
			if err == iterator.Done {
				break
			}
			insertIntoObjectIdMapping(actualObjectMapping, rootNode.GetNotionObjectId(), obj.GetNotionObjectId())
		}

		treeIter := iterator.GetTreeIterator(rootNode)
		for {
			obj, err := treeIter.Next()
			if err == iterator.Done {
				break
			}

			childIter := iterator.GetChildIterator(obj)
			for {
				childObj, err := childIter.Next()
				if err == iterator.Done {
					break
				}
				insertIntoObjectIdMapping(actualObjectMapping, obj.GetNotionObjectId(), childObj.GetNotionObjectId())
			}
		}

		assert.Equal(mockerObj.objectIdMapping, actualObjectMapping)
	})

	t.Run("Error while writing page for given page", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		// mock all required NotionClient functions
		mockedNotionClient.On("GetPageByID", context.Background(), notionclient.PageID("36dac6ee-76e9-4c99-94a9-b0989be3f624")).Return(&notionapi.Page{}, nil)
		mockedRW.On("WritePage", context.Background(), mock.Anything).Return(rw.DataIdentifier(""), errors.New(ERROR_STR))
		treeBuilder := builder.GetExportTreebuilder(context.Background(), mockedNotionClient, mockedRW,
			&builder.TreeRequest{
				PageIdList: []string{"36dac6ee-76e9-4c99-94a9-b0989be3f624"},
			})
		err := treeBuilder.BuildTree(context.Background())
		assert.NotNil(err)
	})

	t.Run("Error while writing database with given database", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		// mock all required NotionClient functions
		mockedNotionClient.On("GetDatabaseByID", context.Background(), notionclient.DatabaseID("36dac6ee-76e9-4c99-94a9-b0989be3f624")).Return(&notionapi.Database{}, nil)
		mockedRW.On("WriteDatabase", context.Background(), mock.Anything).Return(rw.DataIdentifier(""), errors.New(ERROR_STR))

		treeBuilder := builder.GetExportTreebuilder(context.Background(), mockedNotionClient, mockedRW,
			&builder.TreeRequest{
				DatabaseIdList: []string{"36dac6ee-76e9-4c99-94a9-b0989be3f624"},
			})
		err := treeBuilder.BuildTree(context.Background())
		assert.NotNil(err)
	})

	t.Run("Build tree for given page and database", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedNotionClient := mocks.NewNotionClient(t)

		// mock all ReaderWriter functions
		mockedRW.On("WriteDatabase", context.Background(), mock.Anything).Return(rw.DataIdentifier(uuid.New().String()), nil)
		mockedRW.On("WritePage", context.Background(), mock.Anything).Return(rw.DataIdentifier(uuid.New().String()), nil)
		mockedRW.On("WriteBlock", context.Background(), mock.Anything).Return(rw.DataIdentifier(uuid.New().String()), nil)

		// mock all required NotionClient functions
		mockerObj := getMocker(mockedNotionClient)
		mockerObj.createMappings(t, SPECIFIC_PAGE_DATABASE_TREE)
		mockerObj.mockNotionClientFunctions()
		treeBuilder := builder.GetExportTreebuilder(context.Background(), mockedNotionClient, mockedRW,
			&builder.TreeRequest{
				PageIdList:     []string{"36dac6ee-76e9-4c99-94a9-b0989be3f624"},
				DatabaseIdList: []string{"db770044-b760-402e-862a-50fef8d6b5d9"},
			})

		err := treeBuilder.BuildTree(context.Background())
		assert.Nil(err)
		rootNode, err := treeBuilder.GetRootNode()
		assert.NotNil(rootNode)
		assert.Nil(err)

		actualObjectMapping := make(map[string]map[string]bool, 0)
		childIter := iterator.GetChildIterator(rootNode)
		for {
			obj, err := childIter.Next()
			if err == iterator.Done {
				break
			}
			insertIntoObjectIdMapping(actualObjectMapping, rootNode.GetNotionObjectId(), obj.GetNotionObjectId())
		}

		treeIter := iterator.GetTreeIterator(rootNode)
		for {
			obj, err := treeIter.Next()
			if err == iterator.Done {
				break
			}

			childIter := iterator.GetChildIterator(obj)
			for {
				childObj, err := childIter.Next()
				if err == iterator.Done {
					break
				}
				insertIntoObjectIdMapping(actualObjectMapping, obj.GetNotionObjectId(), childObj.GetNotionObjectId())
			}
		}

		assert.Equal(mockerObj.objectIdMapping, actualObjectMapping)
	})
}
