package builder

import (
	"context"
	"errors"

	"github.com/jomei/notionapi"
	"github.com/sawantshivaji1997/notionbackup/src/notionclient"
	"github.com/sawantshivaji1997/notionbackup/src/rw"
	"github.com/sawantshivaji1997/notionbackup/src/tree/node"
)

const (
	PARENT_TYPE_WORKSPACE = notionapi.ParentType("workspace")
	PARENT_TYPE_DATABASE  = notionapi.ParentType("database_id")
	PARENT_TYPE_PAGE      = notionapi.ParentType("page_id")
	CHILD_TYPE_PAGE       = notionapi.BlockType("child_page")
	CHILD_TYPE_DATABASE   = notionapi.BlockType("child_database")
	PROPERTY_TYPE_TITLE   = notionapi.PropertyType("title")
)

type ExportTreeBuilder struct {
	notionClient               notionclient.NotionClient
	rw                         rw.ReaderWriter
	err                        error
	rootNode                   *node.Node
	pageId2PageNodeMap         map[string]*node.Node
	databaseId2DatabaseNodeMap map[string]*node.Node
	databaseId2PageListMap     map[string][]string
	nodeStack                  stack
	request                    *TreeBuilderRequest
}

func GetExportTreebuilder(ctx context.Context,
	notionClient notionclient.NotionClient, rw rw.ReaderWriter,
	request *TreeBuilderRequest) TreeBuilder {
	return &ExportTreeBuilder{
		notionClient:               notionClient,
		rw:                         rw,
		err:                        errors.New("tree was never built"),
		rootNode:                   nil,
		pageId2PageNodeMap:         make(map[string]*node.Node),
		databaseId2DatabaseNodeMap: make(map[string]*node.Node),
		databaseId2PageListMap:     make(map[string][]string),
		nodeStack:                  make(stack, 0),
		request:                    request,
	}
}

// Check if Parent type is workspace
func (builderObj *ExportTreeBuilder) isParentWorkspace(
	parent *notionapi.Parent) bool {
	return parent.Type == PARENT_TYPE_WORKSPACE
}

// Helper function to add database ID to Page Id list mapping
func (builderObj *ExportTreeBuilder) addDatabaseIdToPageMapping(
	page *notionapi.Page) {
	if page.Parent.Type != PARENT_TYPE_DATABASE {
		return
	}

	databaseId := page.Parent.DatabaseID
	if pageList, found := builderObj.
		databaseId2PageListMap[databaseId.String()]; found {
		builderObj.databaseId2PageListMap[databaseId.String()] =
			append(pageList, page.ID.String())
		return
	}

	builderObj.databaseId2PageListMap[databaseId.String()] =
		[]string{page.ID.String()}
}

// Create node object for given page and add it's children to created page node
// object
func (builderObj *ExportTreeBuilder) addPage(ctx context.Context,
	parentNode *node.Node, pageId string) error {
	var pageNode *node.Node

	if nodeObj, found := builderObj.pageId2PageNodeMap[pageId]; found {
		pageNode = nodeObj
		delete(builderObj.pageId2PageNodeMap, pageId)
	} else {
		page, err := builderObj.notionClient.GetPageByID(ctx,
			notionclient.PageID(pageId))
		if err != nil {
			return err
		}
		nodeObj, err := node.CreatePageNode(ctx, page, builderObj.rw)
		if err != nil {
			return err
		}
		pageNode = nodeObj
	}

	parentNode.AddChild(pageNode)
	builderObj.nodeStack.Push(pageNode)
	return nil
}

// Query all the blocks of the page and add them to given node i.e. parentNode
func (builderObj *ExportTreeBuilder) queryAndAddPageChildren(
	ctx context.Context, parentNode *node.Node, pageId string) error {
	cursor := notionapi.Cursor("")
	for {
		var blocks []notionapi.Block
		var err error
		blocks, cursor, err = builderObj.notionClient.GetPageBlocks(
			ctx, notionclient.PageID(pageId), cursor)

		if err != nil {
			return err
		}

		for _, block := range blocks {
			err = builderObj.addBlock(ctx, parentNode, block)
			if err != nil {
				return err
			}
		}

		if cursor == "" {
			break
		}
	}

	return nil
}

// Create node object for given database and add it's children to created
// database node object
func (builderObj *ExportTreeBuilder) addDatabase(ctx context.Context,
	parentNode *node.Node, databaseId string) error {
	var databaseNode *node.Node

	if nodeObj, found := builderObj.
		databaseId2DatabaseNodeMap[databaseId]; found {
		databaseNode = nodeObj
		delete(builderObj.databaseId2DatabaseNodeMap, databaseId)
	} else {
		database, err := builderObj.notionClient.
			GetDatabaseByID(ctx, notionclient.DatabaseID(databaseId))
		if err != nil {
			return err
		}

		nodeObj, err := node.CreateDatabaseNode(ctx, database, builderObj.rw)
		if err != nil {
			return err
		}
		databaseNode = nodeObj
	}

	parentNode.AddChild(databaseNode)
	builderObj.nodeStack.Push(databaseNode)
	return nil
}

// Query all the pages of the given database and add them to the given node i.e
// parentNode
func (builderObj *ExportTreeBuilder) queryAndAddDatabaseChildren(
	ctx context.Context, parentNode *node.Node, databaseId string) error {

	if pageIdList, found := builderObj.databaseId2PageListMap[databaseId]; found {
		for _, pageId := range pageIdList {
			err := builderObj.addPage(ctx, parentNode, pageId)

			if err != nil {
				return err
			}
		}

		delete(builderObj.databaseId2PageListMap, databaseId)
		return nil
	}

	cursor := notionapi.Cursor("")
	for {
		var pages []notionapi.Page
		var err error
		pages, cursor, err = builderObj.notionClient.GetDatabasePages(ctx,
			notionclient.DatabaseID(databaseId), cursor)

		if err != nil {
			return err
		}

		for _, page := range pages {
			pageNode, err := node.CreatePageNode(ctx, &page, builderObj.rw)
			if err != nil {
				return err
			}
			parentNode.AddChild(pageNode)
			builderObj.nodeStack.Push(pageNode)
		}

		if cursor == "" {
			break
		}
	}

	return nil
}

// Create node object for given block and add it's children to created
// block node object
func (builderObj *ExportTreeBuilder) addBlock(ctx context.Context,
	parentNode *node.Node, block notionapi.Block) error {
	blockNode, err := node.CreateBlockNode(ctx, block, builderObj.rw)

	if err != nil {
		return err
	}
	parentNode.AddChild(blockNode)

	if block.GetType() == CHILD_TYPE_DATABASE {
		return builderObj.addDatabase(ctx, blockNode, block.GetID().String())
	}

	if block.GetType() == CHILD_TYPE_PAGE {
		return builderObj.addPage(ctx, blockNode, block.GetID().String())
	}

	if block.GetHasChildren() {
		builderObj.nodeStack.Push(blockNode)
	}

	return nil
}

// Query all the child blocks of the given block and add them to the given node
// i.e. parentNode
func (builderObj *ExportTreeBuilder) queryAndAddBlockChildren(
	ctx context.Context, parentNode *node.Node, blockId string) error {
	cursor := notionapi.Cursor("")
	for {
		var blocks []notionapi.Block
		var err error
		blocks, cursor, err = builderObj.notionClient.GetChildBlocksOfBlock(
			ctx, notionclient.BlockID(blockId), cursor)

		if err != nil {
			return err
		}

		for _, block := range blocks {
			err = builderObj.addBlock(ctx, parentNode, block)
			if err != nil {
				return err
			}
		}

		if cursor.String() == "" {
			break
		}
	}

	return nil
}

// This function will fetch all the pages. Pages which belong to workspace will
// be added to tree and stack and rest of them will be cached which will be
// later used while building the tree.
func (builderObj *ExportTreeBuilder) addWorkspacePages(ctx context.Context,
	parentNode *node.Node) error {
	cursor := notionapi.Cursor("")
	for {
		var pages []notionapi.Page
		var err error
		pages, cursor, err = builderObj.notionClient.GetAllPages(ctx, cursor)

		if err != nil {
			return err
		}

		for _, page := range pages {
			if builderObj.isParentWorkspace(&page.Parent) {
				pageNode, err := node.CreatePageNode(ctx, &page, builderObj.rw)
				if err != nil {
					return err
				}
				parentNode.AddChild(pageNode)
				builderObj.nodeStack.Push(pageNode)
				continue
			}

			// cache for later use
			pageNode, err := node.CreatePageNode(ctx, &page, builderObj.rw)
			if err != nil {
				return nil
			}

			builderObj.addDatabaseIdToPageMapping(&page)
			builderObj.pageId2PageNodeMap[string(page.ID)] = pageNode
		}

		if cursor == "" {
			break
		}
	}
	return nil
}

// This function will fetch all the databases. Databases which belong to
// workspace will be added to tree and stack and rest of them will be cached
// which will be later used while building the tree.
func (builderObj *ExportTreeBuilder) addWorkspaceDatabases(ctx context.Context,
	parentNode *node.Node) error {
	cursor := notionapi.Cursor("")
	for {
		var databases []notionapi.Database
		var err error
		databases, cursor, err = builderObj.notionClient.GetAllDatabases(
			ctx, cursor)
		if err != nil {
			return err
		}

		for _, database := range databases {
			if builderObj.isParentWorkspace(&database.Parent) {
				databaseNode, err := node.CreateDatabaseNode(
					ctx, &database, builderObj.rw)
				if err != nil {
					return err
				}
				parentNode.AddChild(databaseNode)
				builderObj.nodeStack.Push(databaseNode)
				continue
			}

			// cache for later use
			databaseNode, err := node.CreateDatabaseNode(
				ctx, &database, builderObj.rw)
			if err != nil {
				return nil
			}

			builderObj.databaseId2DatabaseNodeMap[string(database.ID)] = databaseNode
		}

		if cursor == "" {
			break
		}
	}

	return nil
}

// Takes node out of stack, query it's children and add thems to tree
// This continues until stack gets empty
func (builderObj *ExportTreeBuilder) buildTreeUntilStackEmpty(
	ctx context.Context) error {
	for {
		object, err := builderObj.nodeStack.Pop()
		if err == StackEmpty {
			break
		}

		if object.GetNodeType() == node.PAGE {
			err = builderObj.queryAndAddPageChildren(
				ctx, object, object.GetNotionObjectId())
		} else if object.GetNodeType() == node.DATABASE {
			err = builderObj.queryAndAddDatabaseChildren(
				ctx, object, object.GetNotionObjectId())
		} else if object.GetNodeType() == node.BLOCK {
			err = builderObj.queryAndAddBlockChildren(
				ctx, object, object.GetNotionObjectId())
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// This function will build the tree for whole workspace depending on Databases
// and Pages the token has access to
func (builderObj *ExportTreeBuilder) buildTreeForWorkspace(
	ctx context.Context) error {
	rootNode := node.CreateRootNode()

	err := builderObj.addWorkspacePages(ctx, rootNode)
	if err != nil {
		return err
	}

	err = builderObj.addWorkspaceDatabases(ctx, rootNode)
	if err != nil {
		return err
	}

	err = builderObj.buildTreeUntilStackEmpty(ctx)
	if err != nil {
		return err
	}

	builderObj.rootNode = rootNode
	return nil
}

// This function will build the tree for given PageIds and DatabaseIds and its
// children
func (builderObj *ExportTreeBuilder) buildTreeForGivenObjectIds(
	ctx context.Context) error {
	rootNode := node.CreateRootNode()

	for _, pageId := range builderObj.request.PageIdList {
		err := builderObj.addPage(ctx, rootNode, pageId)
		if err != nil {
			return err
		}
	}

	for _, databaseId := range builderObj.request.DatabaseIdList {
		err := builderObj.addDatabase(ctx, rootNode, databaseId)
		if err != nil {
			return err
		}
	}

	err := builderObj.buildTreeUntilStackEmpty(ctx)
	if err != nil {
		return err
	}

	builderObj.rootNode = rootNode
	return nil
}

// Build the tree for the given config
func (builderObj *ExportTreeBuilder) BuildTree(ctx context.Context) error {
	if builderObj.rootNode != nil {
		return nil
	}

	// If no PageIDs and DatabaseIDs provided, build tree with all pages and
	// databases from workspace which user has access
	if len(builderObj.request.DatabaseIdList) == 0 &&
		len(builderObj.request.PageIdList) == 0 {
		builderObj.err = builderObj.buildTreeForWorkspace(ctx)
	} else {
		builderObj.err = builderObj.buildTreeForGivenObjectIds(ctx)
	}

	if builderObj.err != nil && builderObj.rw.CleanUp(ctx) != nil {
		// Add logging or printing statement
	}

	return builderObj.err
}

// Get the root node of the tree
func (builderObj *ExportTreeBuilder) GetRootNode() (*node.Node, error) {
	return builderObj.rootNode, builderObj.err
}
