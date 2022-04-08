package notionclient

import (
	"context"

	"github.com/jomei/notionapi"
)

type PageID string
type PageName string
type DatabaseID string
type DatabaseName string
type BlockID string
type Token string

const (
	DEFAULT_PAGE_SIZE = 100
)

type ObjectType int

const (
	UNKNOWN  ObjectType = 0
	DATABASE            = 1
	PAGE                = 2
	BLOCK               = 3
)

type (
	NewClient func(notionapi.Token, ...notionapi.ClientOption) *notionapi.Client
)

// Currently, Filter type in github.com/jomei/notionapi has type interface so
// it's concrete implementation here
type Filter struct {
	Value    string `json:"value"`
	Property string `json:"property"`
}

type NotionClient interface {
	GetAllPages(context.Context) ([]notionapi.Page, error)
	GetAllDatabases(context.Context) ([]notionapi.Database, error)
	GetPagesByName(context.Context, PageName) ([]notionapi.Page, error)
	GetDatabasesByName(context.Context, DatabaseName) ([]notionapi.Database, error)
}

type NotionApiClient struct {
	Client *notionapi.Client
}

// Function to get NotionApiClient instance
func GetNotionApiClient(ctx context.Context, token notionapi.Token, newClient NewClient) NotionClient {
	return &NotionApiClient{
		Client: newClient(token),
	}
}

// Helper function for searching the required objects i.e. pages and databases
// with given query parameter
func (c *NotionApiClient) search(ctx context.Context, objectType ObjectType, cursor notionapi.Cursor, query string) (*notionapi.SearchResponse, error) {
	req := &notionapi.SearchRequest{
		Query: query,
		Filter: Filter{
			Value:    "page",
			Property: "object",
		},
		PageSize:    100,
		StartCursor: notionapi.Cursor(cursor),
	}

	resp, err := c.Client.Search.Do(ctx, req)
	return resp, err
}

// Helper function to get all pages matching the given page name
func (c *NotionApiClient) getPages(ctx context.Context, name PageName) ([]notionapi.Page, error) {
	pages := []notionapi.Page{}
	cursor := notionapi.Cursor("")
	for {
		resp, err := c.search(ctx, PAGE, cursor, string(name))
		if err != nil {
			return nil, err
		}

		for _, result := range resp.Results {
			page := result.(*notionapi.Page)
			pages = append(pages, *page)
		}

		if resp.HasMore {
			cursor = resp.NextCursor
		} else {
			break
		}
	}
	return pages, nil
}

// Get all pages. Passing empty name would mean fetching all the pages from
// workspace
func (c *NotionApiClient) GetAllPages(ctx context.Context) ([]notionapi.Page, error) {
	return c.getPages(ctx, "")
}

// Get all pages matching the given page name
func (c *NotionApiClient) GetPagesByName(ctx context.Context, name PageName) ([]notionapi.Page, error) {
	return c.getPages(ctx, name)
}

// Helper function to get all databases matching the given database name
func (c *NotionApiClient) getDatabases(ctx context.Context, name DatabaseName) ([]notionapi.Database, error) {
	databases := []notionapi.Database{}
	cursor := notionapi.Cursor("")
	for {
		resp, err := c.search(ctx, PAGE, cursor, string(name))
		if err != nil {
			return nil, err
		}

		for _, result := range resp.Results {
			database := result.(*notionapi.Database)
			databases = append(databases, *database)
		}

		if resp.HasMore {
			cursor = resp.NextCursor
		} else {
			break
		}

	}
	return databases, nil
}

// Get all databases. Passing empty name would mean fetching all the databases from
// workspace
func (c *NotionApiClient) GetAllDatabases(ctx context.Context) ([]notionapi.Database, error) {
	return c.getDatabases(ctx, "")
}

// Get all databases matching the given page name
func (c *NotionApiClient) GetDatabasesByName(ctx context.Context, name DatabaseName) ([]notionapi.Database, error) {
	return c.getDatabases(ctx, name)
}
