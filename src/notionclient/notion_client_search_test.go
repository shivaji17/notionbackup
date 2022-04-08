package notionclient_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jomei/notionapi"
	"github.com/sawantshivaji1997/notionbackup/src/notionclient"
	"github.com/sawantshivaji1997/notionbackup/src/utils"
	"github.com/stretchr/testify/assert"
)

const (
	ERROR_STR                             = "error occurred"
	TEST_DATA_PATH                        = "./../../testdata/"
	SEARCH_ALL_PAGES_JSON                 = TEST_DATA_PATH + "search/search_all_pages.json"
	SEARCH_PAGES_WITH_PAGINATION_JSON     = TEST_DATA_PATH + "search/search_pages_with_pagination.json"
	SEARCH_PAGES_WITH_NAME_JSON           = TEST_DATA_PATH + "search/search_pages_with_name.json"
	SEARCH_ALL_DATABASES_JSON             = TEST_DATA_PATH + "search/search_all_databases.json"
	SEARCH_DATABASES_WITH_PAGINATION_JSON = TEST_DATA_PATH + "search/search_databases_with_pagination.json"
	SEARCH_DATABASES_WITH_NAME_JSON       = TEST_DATA_PATH + "search/search_databases_with_name.json"
	EMPTY_SEARCH_RESULT                   = TEST_DATA_PATH + "search/empty_search_result.json"
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

// Mocking Searching service from github.com/jomei/notionapi
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

	jsonBytes, err := utils.ReadJsonFile(mockFilePath)
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

	assert.NotNil(t, searchResponse2)

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
	}{
		{
			name:        "Get all Pages",
			filePath:    SEARCH_ALL_PAGES_JSON,
			wantErr:     false,
			emptyResult: false,
			err:         nil,
		},
		{
			name:        "Get pages with pagination",
			filePath:    SEARCH_PAGES_WITH_PAGINATION_JSON,
			wantErr:     false,
			emptyResult: false,
			err:         nil,
		},
		{
			name:        "Error in fetching pages",
			filePath:    "",
			wantErr:     true,
			emptyResult: true,
			err:         errors.New(ERROR_STR),
		},
		{
			name:        "Get empty Page list",
			filePath:    EMPTY_SEARCH_RESULT,
			wantErr:     false,
			emptyResult: true,
			err:         nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := GetMockedSearchService(t, test.filePath, test.err)
			pages, err := client.GetAllPages(context.Background())
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
			name:        "Get existing database",
			pageName:    "Page-1",
			filePath:    SEARCH_PAGES_WITH_NAME_JSON,
			wantErr:     false,
			emptyResult: false,
			err:         nil,
		},
		{
			name:        "Get non-existing database",
			pageName:    "Page-2",
			filePath:    EMPTY_SEARCH_RESULT,
			wantErr:     false,
			emptyResult: true,
			err:         nil,
		},
		{
			name:        "Error while fetching database",
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
			pages, err := client.GetPagesByName(context.Background(), notionclient.PageName(test.pageName))
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
			databases, err := client.GetAllDatabases(context.Background())
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
			databases, err := client.GetDatabasesByName(context.Background(), notionclient.DatabaseName(test.databaseName))
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
