package notionclient_test

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/jomei/notionapi"
	"github.com/sawantshivaji1997/notionbackup/src/notionclient"
	"github.com/sawantshivaji1997/notionbackup/src/utils"
	"github.com/stretchr/testify/assert"
)

const (
	ERROR_STR                             = "error occurred"
	TEST_DATA_PATH                        = "./../../testdata/notionclient/"
	SEARCH_PATH                           = TEST_DATA_PATH + "search/"
	SEARCH_ALL_PAGES_JSON                 = SEARCH_PATH + "search_all_pages.json"
	SEARCH_PAGES_WITH_PAGINATION_JSON     = SEARCH_PATH + "search_pages_with_pagination.json"
	SEARCH_PAGES_WITH_NAME_JSON           = SEARCH_PATH + "search_pages_with_name.json"
	SEARCH_ALL_DATABASES_JSON             = SEARCH_PATH + "search_all_databases.json"
	SEARCH_DATABASES_WITH_PAGINATION_JSON = SEARCH_PATH + "search_databases_with_pagination.json"
	SEARCH_DATABASES_WITH_NAME_JSON       = SEARCH_PATH + "search_databases_with_name.json"
	EMPTY_SEARCH_RESULT                   = SEARCH_PATH + "empty_search_result.json"

	PAGE_PATH                = TEST_DATA_PATH + "page/"
	PAGE_JSON                = PAGE_PATH + "page.json"
	PAGE_NOT_EXIST_ERROR_STR = "Page does not exist"

	DATABASE_PATH                = TEST_DATA_PATH + "database/"
	DATABASE_JSON                = DATABASE_PATH + "database.json"
	DATABASE_QUERY_RESPONSE_JSON = DATABASE_PATH + "database_query_response.json"
	DATABASE_NOT_EXIST_ERROR_STR = "Database does not exist"

	BLOCK_PATH       = TEST_DATA_PATH + "block/"
	PAGE_BLOCKS_JSON = BLOCK_PATH + "page_blocks.json"
	BLOCKS_JSON      = BLOCK_PATH + "block.json"
)

// Mocking the NewClient from github.com/jomei/notionapi
func newMockedClient(token notionapi.Token, opt ...notionapi.ClientOption) *notionapi.Client {
	return &notionapi.Client{}
}

func TestGetNotionClient(t *testing.T) {
	t.Run("Get Client with valid parameters", func(t *testing.T) {
		client := notionclient.GetNotionApiClient(context.Background(), "asdasd", newMockedClient)
		assert.NotNil(t, client)
	})
}

// Mocking the SearchService from github.com/jomei/notionapi
type MockedSearchService struct {
	response  *notionapi.SearchResponse
	response2 *notionapi.SearchResponse
	err       error
}

func GetMockedSearchService(t *testing.T, mockFilePath string, err error) *notionclient.NotionApiClient {
	if err != nil {
		return &notionclient.NotionApiClient{
			Client: &notionapi.Client{
				Search: &MockedSearchService{
					response:  nil,
					response2: nil,
					err:       err,
				},
			},
		}
	}

	jsonBytes, err := ioutil.ReadFile(mockFilePath)
	if err != nil {
		t.Fatal(err)
	}

	searchResponse, err := utils.ParseSearchResponseJsonString(jsonBytes)

	if err != nil {
		t.Fatal(err)
	}

	var searchResponse2 notionapi.SearchResponse
	if searchResponse.HasMore {
		searchResponse2, _ := utils.ParseSearchResponseJsonString(jsonBytes)
		searchResponse2.HasMore = false
		searchResponse2.NextCursor = notionapi.Cursor("")
	}

	return &notionclient.NotionApiClient{
		Client: &notionapi.Client{
			Search: &MockedSearchService{
				response:  searchResponse,
				response2: &searchResponse2,
				err:       nil,
			},
		},
	}
}

func (srv *MockedSearchService) Do(ctx context.Context, req *notionapi.SearchRequest) (*notionapi.SearchResponse, error) {
	if req.StartCursor != "" {
		return srv.response2, srv.err
	}

	return srv.response, srv.err
}

func TestGetAllPages(t *testing.T) {

	tests := []struct {
		name        string
		filePath    string
		wantErr     bool
		emptyResult bool
		err         error
		cursorEmpty bool
	}{
		{
			name:        "Get all Pages",
			filePath:    SEARCH_ALL_PAGES_JSON,
			wantErr:     false,
			emptyResult: false,
			err:         nil,
			cursorEmpty: true,
		},
		{
			name:        "Get pages with pagination",
			filePath:    SEARCH_PAGES_WITH_PAGINATION_JSON,
			wantErr:     false,
			emptyResult: false,
			err:         nil,
			cursorEmpty: false,
		},
		{
			name:        "Error in fetching pages",
			filePath:    "",
			wantErr:     true,
			emptyResult: true,
			err:         errors.New(ERROR_STR),
			cursorEmpty: true,
		},
		{
			name:        "Get empty Page list",
			filePath:    EMPTY_SEARCH_RESULT,
			wantErr:     false,
			emptyResult: true,
			err:         nil,
			cursorEmpty: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := GetMockedSearchService(t, test.filePath, test.err)
			pages, cursor, err := client.GetAllPages(context.Background(), notionapi.Cursor(""))
			if test.wantErr {
				assert.Nil(t, pages)
				assert.NotNil(t, err)
				assert.Empty(t, cursor)
			} else {
				if test.emptyResult {
					assert.Empty(t, pages)
				} else {
					assert.NotEmpty(t, pages)
					if test.cursorEmpty {
						assert.Empty(t, cursor)
					} else {
						assert.NotEmpty(t, cursor)
					}
				}
				assert.Nil(t, err)
			}
		})
	}
}

func TestGetPagesByName(t *testing.T) {
	tests := []struct {
		name        string
		pageName    string
		filePath    string
		wantErr     bool
		emptyResult bool
		err         error
	}{
		{
			name:        "Get existing pages",
			pageName:    "Page-1",
			filePath:    SEARCH_PAGES_WITH_NAME_JSON,
			wantErr:     false,
			emptyResult: false,
			err:         nil,
		},
		{
			name:        "Get non-existing pages",
			pageName:    "Page-2",
			filePath:    EMPTY_SEARCH_RESULT,
			wantErr:     false,
			emptyResult: true,
			err:         nil,
		},
		{
			name:        "Error while fetching pages",
			pageName:    "Page-1",
			filePath:    SEARCH_PAGES_WITH_NAME_JSON,
			wantErr:     true,
			emptyResult: true,
			err:         errors.New(ERROR_STR),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := GetMockedSearchService(t, test.filePath, test.err)
			pages, _, err := client.GetPagesByName(context.Background(), notionclient.PageName(test.pageName), "")
			if test.wantErr {
				assert.Nil(t, pages)
				assert.NotNil(t, err)
			} else {
				if test.emptyResult {
					assert.Empty(t, pages)
				} else {
					assert.NotEmpty(t, pages)
				}
				assert.Nil(t, err)
			}
		})
	}
}

func TestGetAllDatabases(t *testing.T) {
	tests := []struct {
		name        string
		filePath    string
		wantErr     bool
		emptyResult bool
		err         error
	}{
		{
			name:        "Get all Databases",
			filePath:    SEARCH_ALL_DATABASES_JSON,
			wantErr:     false,
			emptyResult: false,
			err:         nil,
		},
		{
			name:        "Get databases with pagination",
			filePath:    SEARCH_DATABASES_WITH_PAGINATION_JSON,
			wantErr:     false,
			emptyResult: false,
			err:         nil,
		},
		{
			name:        "Error in fetching databases",
			filePath:    "",
			wantErr:     true,
			emptyResult: true,
			err:         errors.New(ERROR_STR),
		},
		{
			name:        "Get empty Database list",
			filePath:    EMPTY_SEARCH_RESULT,
			wantErr:     false,
			emptyResult: true,
			err:         nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := GetMockedSearchService(t, test.filePath, test.err)
			databases, _, err := client.GetAllDatabases(context.Background(), "")
			if test.wantErr {
				assert.Nil(t, databases)
				assert.NotNil(t, err)
			} else {
				if test.emptyResult {
					assert.Empty(t, databases)
				} else {
					assert.NotEmpty(t, databases)
				}
				assert.Nil(t, err)
			}
		})
	}
}

func TestGetDatabasesByName(t *testing.T) {
	tests := []struct {
		name         string
		databaseName string
		filePath     string
		wantErr      bool
		emptyResult  bool
		err          error
	}{
		{
			name:         "Get existing database",
			databaseName: "Database-1",
			filePath:     SEARCH_DATABASES_WITH_NAME_JSON,
			wantErr:      false,
			emptyResult:  false,
			err:          nil,
		},
		{
			name:         "Get non-existing database",
			databaseName: "Database-2",
			filePath:     EMPTY_SEARCH_RESULT,
			wantErr:      false,
			emptyResult:  true,
			err:          nil,
		},
		{
			name:         "Error while fetching database",
			databaseName: "Database-1",
			filePath:     SEARCH_DATABASES_WITH_NAME_JSON,
			wantErr:      true,
			emptyResult:  true,
			err:          errors.New(ERROR_STR),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := GetMockedSearchService(t, test.filePath, test.err)
			databases, _, err := client.GetDatabasesByName(context.Background(), notionclient.DatabaseName(test.databaseName), "")
			if test.wantErr {
				assert.Nil(t, databases)
				assert.NotNil(t, err)
			} else {
				if test.emptyResult {
					assert.Empty(t, databases)
				} else {
					assert.NotEmpty(t, databases)
				}
				assert.Nil(t, err)
			}
		})
	}
}

// Mocking the PageService from github.com/jomei/notionapi
type MockedPageService struct {
	page *notionapi.Page
	err  error
}

func GetMockedPageService(t *testing.T, mockFilePath string, err error) *notionclient.NotionApiClient {
	if err != nil {
		return &notionclient.NotionApiClient{
			Client: &notionapi.Client{
				Page: &MockedPageService{
					page: nil,
					err:  err,
				},
			},
		}
	}

	jsonBytes, err := ioutil.ReadFile(mockFilePath)
	if err != nil {
		t.Fatal(err)
	}

	page, err := utils.ParsePageJsonString(jsonBytes)

	if err != nil {
		t.Fatal(err)
	}

	return &notionclient.NotionApiClient{
		Client: &notionapi.Client{
			Page: &MockedPageService{
				page: page,
				err:  nil,
			},
		},
	}
}

func (srv *MockedPageService) Get(ctx context.Context, id notionapi.PageID) (*notionapi.Page, error) {
	return srv.page, srv.err
}

func (srv *MockedPageService) Create(ctx context.Context, req *notionapi.PageCreateRequest) (*notionapi.Page, error) {
	// TODO
	return nil, nil
}

func (srv *MockedPageService) Update(ctx context.Context, id notionapi.PageID, req *notionapi.PageUpdateRequest) (*notionapi.Page, error) {
	// TODO
	return nil, nil
}

func TestGetPageByID(t *testing.T) {
	tests := []struct {
		name        string
		pageid      string
		filePath    string
		wantErr     bool
		emptyResult bool
		err         error
	}{
		{
			name:        "Get existing Page",
			pageid:      "05034203-2870-4bc8-b1f9-22c0ae6e56ba",
			filePath:    PAGE_JSON,
			wantErr:     false,
			emptyResult: false,
			err:         nil,
		},
		{
			name:        "Get non-existing Page",
			pageid:      "05034203-2870-4bc8-b1f9-22c0ae6e56aa",
			filePath:    PAGE_JSON,
			wantErr:     true,
			emptyResult: true,
			err:         errors.New(PAGE_NOT_EXIST_ERROR_STR),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := GetMockedPageService(t, test.filePath, test.err)
			page, err := client.GetPageByID(context.Background(), notionclient.PageID(test.pageid))
			if test.wantErr {
				assert.Nil(t, page)
				assert.NotNil(t, err)
			} else {
				if test.emptyResult {
					assert.Empty(t, page)
				} else {
					assert.NotEmpty(t, page)
				}
				assert.Nil(t, err)
			}
		})

	}
}

// Mocking the DatabaseService from github.com/jomei/notionapi
type MockedDatabaseService struct {
	database              *notionapi.Database
	databaseQueryResponse *notionapi.DatabaseQueryResponse
	err                   error
}

func GetMockedDatabaseService(t *testing.T, databaseFilePath string, databaseQueryResponseFile string, err error) *notionclient.NotionApiClient {
	if err != nil {
		return &notionclient.NotionApiClient{
			Client: &notionapi.Client{
				Database: &MockedDatabaseService{
					database:              nil,
					databaseQueryResponse: nil,
					err:                   err,
				},
			},
		}
	}

	jsonBytes, err := ioutil.ReadFile(databaseFilePath)
	if err != nil {
		t.Fatal(err)
	}

	database, err := utils.ParseDatabaseJsonString(jsonBytes)

	if err != nil {
		t.Fatal(err)
	}

	jsonBytes2, err := ioutil.ReadFile(databaseQueryResponseFile)
	if err != nil {
		t.Fatal(err)
	}

	databaseQueryResponse := &notionapi.DatabaseQueryResponse{}
	err = json.Unmarshal(jsonBytes2, &databaseQueryResponse)
	if err != nil {
		t.Fatal(err)
	}

	return &notionclient.NotionApiClient{
		Client: &notionapi.Client{
			Database: &MockedDatabaseService{
				database:              database,
				databaseQueryResponse: databaseQueryResponse,
				err:                   nil,
			},
		},
	}
}

func (srv *MockedDatabaseService) Get(ctx context.Context, id notionapi.DatabaseID) (*notionapi.Database, error) {
	return srv.database, srv.err
}

func (srv *MockedDatabaseService) List(ctx context.Context, pagination *notionapi.Pagination) (*notionapi.DatabaseListResponse, error) {
	// Not needed. Just keeping as a placeholder for DatabaseService interface
	// List REST API call is deprecated by Notion
	return nil, nil
}

func (srv *MockedDatabaseService) Query(ctx context.Context, id notionapi.DatabaseID, req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
	// TODO
	return srv.databaseQueryResponse, srv.err
}

func (srv *MockedDatabaseService) Update(ctx context.Context, id notionapi.DatabaseID, req *notionapi.DatabaseUpdateRequest) (*notionapi.Database, error) {
	// TODO
	return nil, nil
}

func (srv *MockedDatabaseService) Create(ctx context.Context, req *notionapi.DatabaseCreateRequest) (*notionapi.Database, error) {
	// TODO
	return nil, nil
}

func TestGetDatabaseByID(t *testing.T) {
	tests := []struct {
		name                     string
		databaseid               string
		databaseFilePath         string
		databaseQueryRspFilePath string
		wantErr                  bool
		emptyResult              bool
		err                      error
	}{
		{
			name:                     "Get existing Database",
			databaseid:               "db770044-b760-402e-862a-50fef8d6b5d9",
			databaseFilePath:         DATABASE_JSON,
			databaseQueryRspFilePath: DATABASE_QUERY_RESPONSE_JSON,
			wantErr:                  false,
			emptyResult:              false,
			err:                      nil,
		},
		{
			name:                     "Get non-existing Database",
			databaseid:               "3caeee7e-2774-4d17-a911-a17b5d1b92da",
			databaseFilePath:         DATABASE_JSON,
			databaseQueryRspFilePath: DATABASE_QUERY_RESPONSE_JSON,
			wantErr:                  true,
			emptyResult:              true,
			err:                      errors.New(DATABASE_NOT_EXIST_ERROR_STR),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := GetMockedDatabaseService(t, test.databaseFilePath, test.databaseQueryRspFilePath, test.err)
			database, err := client.GetDatabaseByID(context.Background(), notionclient.DatabaseID(test.databaseid))
			if test.wantErr {
				assert.Nil(t, database)
				assert.NotNil(t, err)
			} else {
				if test.emptyResult {
					assert.Empty(t, database)
				} else {
					assert.NotEmpty(t, database)
				}
				assert.Nil(t, err)
			}
		})
	}
}

func TestGetPagesOfDatabase(t *testing.T) {
	tests := []struct {
		name                     string
		databaseid               string
		databaseFilePath         string
		databaseQueryRspFilePath string
		wantErr                  bool
		emptyResult              bool
		err                      error
	}{
		{
			name:                     "Get all pages of database",
			databaseid:               "db770044-b760-402e-862a-50fef8d6b5d9",
			databaseFilePath:         DATABASE_JSON,
			databaseQueryRspFilePath: DATABASE_QUERY_RESPONSE_JSON,
			wantErr:                  false,
			emptyResult:              false,
			err:                      nil,
		},
		{
			name:                     "Get non-existing Database",
			databaseid:               "3caeee7e-2774-4d17-a911-a17b5d1b92da",
			databaseFilePath:         DATABASE_JSON,
			databaseQueryRspFilePath: DATABASE_QUERY_RESPONSE_JSON,
			wantErr:                  true,
			emptyResult:              true,
			err:                      errors.New(DATABASE_NOT_EXIST_ERROR_STR),
		},
		{
			name:                     "Get zero pages of Database",
			databaseid:               "3caeee7e-2774-4d17-a911-a17b5d1b92da",
			databaseFilePath:         DATABASE_JSON,
			databaseQueryRspFilePath: EMPTY_SEARCH_RESULT,
			wantErr:                  false,
			emptyResult:              true,
			err:                      nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := GetMockedDatabaseService(t, test.databaseFilePath, test.databaseQueryRspFilePath, test.err)
			pages, _, err := client.GetPagesOfDatabase(context.Background(), notionclient.DatabaseID(test.databaseid), notionapi.Cursor(""))
			if test.wantErr {
				assert.Nil(t, pages)
				assert.NotNil(t, err)
			} else {
				if test.emptyResult {
					assert.Empty(t, pages)
				} else {
					assert.NotEmpty(t, pages)
				}
				assert.Nil(t, err)
			}
		})

	}
}

// Mocking the BlockService from github.com/jomei/notionapi
type MockedBlockService struct {
	childBlocks *notionapi.GetChildrenResponse
	block       notionapi.Block
	err         error
}

func GetMockedBlockService(t *testing.T, childBlocksFilePath string, blockFilePath string, err error) *notionclient.NotionApiClient {
	if err != nil {
		return &notionclient.NotionApiClient{
			Client: &notionapi.Client{
				Block: &MockedBlockService{
					childBlocks: nil,
					block:       nil,
					err:         err,
				},
			},
		}
	}

	jsonBytes, err := ioutil.ReadFile(childBlocksFilePath)
	if err != nil {
		t.Fatal(err)
	}

	childBlocks := &notionapi.GetChildrenResponse{}
	err = json.Unmarshal(jsonBytes, &childBlocks)
	if err != nil {
		t.Fatal(err)
	}

	jsonBytes2, err := ioutil.ReadFile(blockFilePath)
	if err != nil {
		t.Fatal(err)
	}

	var response map[string]interface{}
	err = json.Unmarshal(jsonBytes2, &response)
	if err != nil {
		t.Fatal(err)
	}

	block, err := utils.DecodeBlockObject(response)
	if err != nil {
		t.Fatal(err)
	}
	return &notionclient.NotionApiClient{
		Client: &notionapi.Client{
			Block: &MockedBlockService{
				childBlocks: childBlocks,
				block:       block,
				err:         nil,
			},
		},
	}
}

func (srv *MockedBlockService) GetChildren(ctx context.Context, id notionapi.BlockID, pagination *notionapi.Pagination) (*notionapi.GetChildrenResponse, error) {
	return srv.childBlocks, srv.err
}

func (srv *MockedBlockService) AppendChildren(ctx context.Context, id notionapi.BlockID, req *notionapi.AppendBlockChildrenRequest) (*notionapi.AppendBlockChildrenResponse, error) {
	// TODO
	return nil, nil
}

func (srv *MockedBlockService) Get(ctx context.Context, id notionapi.BlockID) (notionapi.Block, error) {
	return srv.block, srv.err
}

func (srv *MockedBlockService) Delete(ctx context.Context, id notionapi.BlockID) (notionapi.Block, error) {
	// TODO
	return nil, nil
}

func (srv *MockedBlockService) Update(ctx context.Context, id notionapi.BlockID, request *notionapi.BlockUpdateRequest) (notionapi.Block, error) {
	// TODO
	return nil, nil
}

func TestGetBlocksOfPagesAndChildBlocksOfBlock(t *testing.T) {
	tests := []struct {
		name                string
		blockID             notionapi.BlockID
		childBlocksFilePath string
		blockFilePath       string
		wantErr             bool
		emptyResult         bool
		err                 error
	}{
		{
			name:                "Get child block for Page",
			blockID:             notionapi.BlockID("e50c7b3ae61c4b26a6f96dfef9f74148"),
			childBlocksFilePath: PAGE_BLOCKS_JSON,
			blockFilePath:       BLOCKS_JSON,
			wantErr:             false,
			emptyResult:         false,
			err:                 nil,
		},
		{
			name:                "Get no blocks for Page",
			blockID:             notionapi.BlockID("e50c7b3ae61c4b26a6f96dfef9f74148"),
			childBlocksFilePath: EMPTY_SEARCH_RESULT,
			blockFilePath:       BLOCKS_JSON,
			wantErr:             true,
			emptyResult:         true,
			err:                 errors.New(EMPTY_SEARCH_RESULT),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := GetMockedBlockService(t, test.childBlocksFilePath, test.blockFilePath, test.err)
			childBlocks, _, err := client.GetBlocksOfPages(context.Background(), notionclient.PageID(test.blockID), notionapi.Cursor(""))
			childBlocks2, _, err2 := client.GetChildBlocksOfBlock(context.Background(), notionclient.BlockID(test.blockID), notionapi.Cursor(""))
			if test.wantErr {
				assert.Nil(t, childBlocks)
				assert.NotNil(t, err)
				assert.Nil(t, childBlocks2)
				assert.NotNil(t, err2)
			} else {
				if test.emptyResult {
					assert.Empty(t, childBlocks)
					assert.Empty(t, childBlocks2)
				} else {
					assert.NotEmpty(t, childBlocks)
					assert.NotEmpty(t, childBlocks2)
				}
				assert.Nil(t, err)
				assert.Nil(t, err2)
			}
		})
	}
}

func TestGetBlockByID(t *testing.T) {
	tests := []struct {
		name                string
		blockID             notionapi.BlockID
		childBlocksFilePath string
		blockFilePath       string
		wantErr             bool
		err                 error
	}{
		{
			name:                "Get existing block",
			blockID:             notionapi.BlockID("c38690eb-049b-4e52-b562-2b774f8d3b73"),
			childBlocksFilePath: PAGE_BLOCKS_JSON,
			blockFilePath:       BLOCKS_JSON,
			wantErr:             false,
			err:                 nil,
		},
		{
			name:                "Get non-existing block",
			blockID:             notionapi.BlockID("e50c7b3ae61c4b26a6f96dfef9f74156"),
			childBlocksFilePath: EMPTY_SEARCH_RESULT,
			blockFilePath:       BLOCKS_JSON,
			wantErr:             true,
			err:                 errors.New(EMPTY_SEARCH_RESULT),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := GetMockedBlockService(t, test.childBlocksFilePath, test.blockFilePath, test.err)
			block, err := client.GetBlockByID(context.Background(), notionclient.BlockID(test.blockID))

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
