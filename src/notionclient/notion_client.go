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

type (
	NewClient func(notionapi.Token, ...notionapi.ClientOption) *notionapi.Client
)

type NotionClient interface {
	GetAllPages(context.Context, notionapi.Cursor) ([]notionapi.Page,
		notionapi.Cursor, error)

	GetAllDatabases(context.Context, notionapi.Cursor) ([]notionapi.Database,
		notionapi.Cursor, error)

	GetPagesByName(context.Context, PageName, notionapi.Cursor) ([]notionapi.Page,
		notionapi.Cursor, error)

	GetDatabasesByName(context.Context, DatabaseName,
		notionapi.Cursor) ([]notionapi.Database, notionapi.Cursor, error)

	GetPageByID(context.Context, PageID) (*notionapi.Page, error)

	GetDatabaseByID(context.Context, DatabaseID) (*notionapi.Database, error)

	GetDatabasePages(context.Context, DatabaseID,
		notionapi.Cursor) ([]notionapi.Page, notionapi.Cursor, error)

	GetPageBlocks(context.Context, PageID, notionapi.Cursor) ([]notionapi.Block,
		notionapi.Cursor, error)

	GetChildBlocksOfBlock(context.Context, BlockID,
		notionapi.Cursor) ([]notionapi.Block, notionapi.Cursor, error)

	GetBlockByID(context.Context, BlockID) (notionapi.Block, error)

	CreatePage(context.Context, *notionapi.PageCreateRequest) (*notionapi.Page,
		error)

	CreateDatabase(context.Context, *notionapi.DatabaseCreateRequest) (
		*notionapi.Database, error)

	AppendBlocksToPage(context.Context, PageID,
		*notionapi.AppendBlockChildrenRequest) (
		*notionapi.AppendBlockChildrenResponse, error)

	AppendBlocksToBlock(context.Context, BlockID,
		*notionapi.AppendBlockChildrenRequest) (
		*notionapi.AppendBlockChildrenResponse, error)
}

type NotionApiClient struct {
	Client *notionapi.Client
}

// Function to get NotionApiClient instance
func GetNotionApiClient(ctx context.Context, token notionapi.Token,
	newClient NewClient) NotionClient {
	return &NotionApiClient{
		Client: newClient(token),
	}
}

// Helper function for searching the required objects i.e. pages and databases
// with given query parameter
func (c *NotionApiClient) search(ctx context.Context, objectType string,
	cursor notionapi.Cursor, query string) (*notionapi.SearchResponse, error) {
	req := &notionapi.SearchRequest{
		Query: query,
		Filter: notionapi.SearchFilter{
			Value:    objectType,
			Property: "object",
		},
		PageSize:    100,
		StartCursor: notionapi.Cursor(cursor),
	}

	resp, err := c.Client.Search.Do(ctx, req)
	return resp, err
}

// Helper function to get all pages matching the given page name
func (c *NotionApiClient) getPages(ctx context.Context, name PageName,
	cursor notionapi.Cursor) ([]notionapi.Page, notionapi.Cursor, error) {
	pages := []notionapi.Page{}

	resp, err := c.search(ctx, "page", cursor, string(name))
	if err != nil {
		return nil, "", err
	}

	for _, result := range resp.Results {
		page := result.(*notionapi.Page)
		pages = append(pages, *page)
	}

	var newCursor notionapi.Cursor
	if resp.HasMore {
		newCursor = resp.NextCursor
	} else {
		newCursor = notionapi.Cursor("")
	}

	return pages, newCursor, nil
}

// Get all pages. Passing empty name would mean fetching all the pages from
// workspace
func (c *NotionApiClient) GetAllPages(ctx context.Context,
	cursor notionapi.Cursor) ([]notionapi.Page, notionapi.Cursor, error) {
	return c.getPages(ctx, "" /*PageName*/, cursor)
}

// Get all pages matching the given page name
func (c *NotionApiClient) GetPagesByName(ctx context.Context, name PageName,
	cursor notionapi.Cursor) ([]notionapi.Page, notionapi.Cursor, error) {
	return c.getPages(ctx, name, cursor)
}

// Helper function to get all databases matching the given database name
func (c *NotionApiClient) getDatabases(ctx context.Context, name DatabaseName,
	cursor notionapi.Cursor) ([]notionapi.Database, notionapi.Cursor, error) {
	databases := []notionapi.Database{}

	resp, err := c.search(ctx, "database", cursor, string(name))
	if err != nil {
		return nil, "", err
	}

	for _, result := range resp.Results {
		database := result.(*notionapi.Database)
		databases = append(databases, *database)
	}

	var newCursor notionapi.Cursor
	if resp.HasMore {
		newCursor = resp.NextCursor
	} else {
		newCursor = notionapi.Cursor("")
	}

	return databases, newCursor, nil
}

// Get all databases. Passing empty name would mean fetching all the databases
// from workspace
func (c *NotionApiClient) GetAllDatabases(ctx context.Context,
	cursor notionapi.Cursor) ([]notionapi.Database, notionapi.Cursor, error) {
	return c.getDatabases(ctx, "" /*DatabaseName*/, cursor)
}

// Get all databases matching the given page name
func (c *NotionApiClient) GetDatabasesByName(ctx context.Context,
	name DatabaseName, cursor notionapi.Cursor) ([]notionapi.Database,
	notionapi.Cursor, error) {
	return c.getDatabases(ctx, name, cursor)
}

// Get Page with given PageID
func (c *NotionApiClient) GetPageByID(ctx context.Context,
	id PageID) (*notionapi.Page, error) {
	return c.Client.Page.Get(ctx, notionapi.PageID(id))
}

// Get Database with given DatabaseID
func (c *NotionApiClient) GetDatabaseByID(ctx context.Context,
	id DatabaseID) (*notionapi.Database, error) {
	return c.Client.Database.Get(ctx, notionapi.DatabaseID(id))
}

// Get all pages for given Database
func (c *NotionApiClient) GetDatabasePages(ctx context.Context,
	id DatabaseID, cursor notionapi.Cursor) ([]notionapi.Page,
	notionapi.Cursor, error) {
	queryReq := &notionapi.DatabaseQueryRequest{
		StartCursor: cursor,
		PageSize:    DEFAULT_PAGE_SIZE,
	}

	resp, err := c.Client.Database.Query(ctx, notionapi.DatabaseID(id), queryReq)
	if err != nil {
		return nil, "", err
	}

	pages := []notionapi.Page{}
	pages = append(pages, resp.Results...)

	var newCursor notionapi.Cursor
	if resp.HasMore {
		newCursor = resp.NextCursor
	} else {
		newCursor = notionapi.Cursor("")
	}

	return pages, newCursor, nil
}

// Helper function to get children blocks of given block which can be either
// page or block
func (c *NotionApiClient) getChildBlocks(ctx context.Context,
	id BlockID, cursor notionapi.Cursor) ([]notionapi.Block, notionapi.Cursor,
	error) {
	pagination := &notionapi.Pagination{
		StartCursor: cursor,
		PageSize:    DEFAULT_PAGE_SIZE,
	}

	resp, err := c.Client.Block.GetChildren(ctx, notionapi.BlockID(id),
		pagination)
	if err != nil {
		return nil, "", err
	}

	blocks := []notionapi.Block{}
	for _, block := range resp.Results {
		blocks = append(blocks, block)
	}
	var newCursor notionapi.Cursor
	if resp.HasMore {
		newCursor = notionapi.Cursor(resp.NextCursor)
	} else {
		newCursor = notionapi.Cursor("")
	}

	return blocks, newCursor, nil
}

// Get all child blocks of given page
func (c *NotionApiClient) GetPageBlocks(ctx context.Context, id PageID,
	cursor notionapi.Cursor) ([]notionapi.Block, notionapi.Cursor, error) {
	return c.getChildBlocks(ctx, BlockID(id), cursor)
}

// Get all child blocks of given block
func (c *NotionApiClient) GetChildBlocksOfBlock(ctx context.Context,
	id BlockID, cursor notionapi.Cursor) ([]notionapi.Block, notionapi.Cursor,
	error) {
	return c.getChildBlocks(ctx, id, cursor)
}

// Get block having given ID
func (c *NotionApiClient) GetBlockByID(ctx context.Context,
	id BlockID) (notionapi.Block, error) {
	return c.Client.Block.Get(ctx, notionapi.BlockID(id))
}

// Create a page object
func (c *NotionApiClient) CreatePage(ctx context.Context,
	req *notionapi.PageCreateRequest) (*notionapi.Page, error) {
	return c.Client.Page.Create(ctx, req)
}

// Create a database object
func (c *NotionApiClient) CreateDatabase(ctx context.Context,
	req *notionapi.DatabaseCreateRequest) (*notionapi.Database, error) {
	return c.Client.Database.Create(ctx, req)
}

// Add blocks to given page ID
func (c *NotionApiClient) AppendBlocksToPage(ctx context.Context, pageID PageID,
	req *notionapi.AppendBlockChildrenRequest) (
	*notionapi.AppendBlockChildrenResponse, error) {
	return c.Client.Block.AppendChildren(ctx, notionapi.BlockID(pageID), req)
}

// Add subblocks to given block ID
func (c *NotionApiClient) AppendBlocksToBlock(ctx context.Context,
	blockID BlockID, req *notionapi.AppendBlockChildrenRequest) (
	*notionapi.AppendBlockChildrenResponse, error) {
	return c.Client.Block.AppendChildren(ctx, notionapi.BlockID(blockID), req)
}
