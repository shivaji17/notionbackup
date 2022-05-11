package builder

import (
	"context"
	"errors"

	"github.com/jomei/notionapi"
	"github.com/sawantshivaji1997/notionbackup/src/notionclient"
	"github.com/sawantshivaji1997/notionbackup/src/rw"
	"github.com/sawantshivaji1997/notionbackup/src/tree/iterator"
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
}

func GetExportTreebuilder(ctx context.Context, notionClient notionclient.NotionClient, rw rw.ReaderWriter) TreeBuilder {
	return &ExportTreeBuilder{
		notionClient:               notionClient,
		rw:                         rw,
		err:                        errors.New("tree was never built"),
		rootNode:                   nil,
		pageId2PageNodeMap:         make(map[string]*node.Node),
		databaseId2DatabaseNodeMap: make(map[string]*node.Node),
		databaseId2PageListMap:     make(map[string][]string),
		nodeStack:                  make(stack, 0),
	}
}

// Check if Parent type is workspace
func (builder *ExportTreeBuilder) isParentWorkspace(parent *notionapi.Parent) bool {
	return parent.Type == PARENT_TYPE_WORKSPACE
}

// Helper function to add page name to pageName2PageIdMap
/*func (builder *ExportTreeBuilder) addPageNameToPageIdMap(page *notionapi.Page) error {
	for _, property := range page.Properties {
		if property.GetType() == PROPERTY_TYPE_TITLE {
			titleProperty, ok := property.(notionapi.TitleProperty)
			if !ok {
				return errors.New("cannot find title property from page object")
			}

			if len(titleProperty.Title) == 0 {
				return errors.New("'title' properties are not present in page object")
			}

			builder.pageName2PageIdMap[string(titleProperty.Title[0].PlainText)] = string(page.ID)
			return nil
		}
	}

	return errors.New("failed to find 'title' property from page object")
}*/

// Helper function to add database name to databaseName2DatabaseIdMap
/*func (builder *ExportTreeBuilder) addDatabaseNameToDatabaseIdMap(database *notionapi.Database) error {
	if len(database.Title) == 0 {
		return errors.New("'title' properties are not present in database object: " + string(database.ID))
	}

	builder.databaseName2DatabaseIdMap[string(database.Title[0].PlainText)] = string(database.ID)
	return nil
}*/

// Helper function to add database ID to Page Id list mapping
func (builder *ExportTreeBuilder) addDatabaseIdToPageMapping(page *notionapi.Page) {
	if page.Parent.Type != PARENT_TYPE_DATABASE {
		return
	}

	databaseId := page.Parent.DatabaseID
	if pageList, found := builder.databaseId2PageListMap[databaseId.String()]; found {
		builder.databaseId2PageListMap[databaseId.String()] = append(pageList, page.ID.String())
		return
	}

	builder.databaseId2PageListMap[databaseId.String()] = []string{page.ID.String()}
}

// Helper function to get non-block type parent Node notionObjectId
func (builder *ExportTreeBuilder) getNonBlockTypeParentNotionObjectId(nodeObj *node.Node) string {
	iter := iterator.GetParentIterator(nodeObj)
	for {
		obj, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if obj.GetNodeType() != node.BLOCK {
			return obj.GetNotionObjectId()
		}
	}

	return ""
}

// Create node object for given page and add it's children to created page node
// object
func (builder *ExportTreeBuilder) addPage(ctx context.Context, parentNode *node.Node, pageId string) error {
	var pageNode *node.Node

	if nodeObj, found := builder.pageId2PageNodeMap[pageId]; found {
		pageNode = nodeObj
	} else {
		page, err := builder.notionClient.GetPageByID(ctx, notionclient.PageID(pageId))
		if err != nil {
			return err
		}
		nodeObj, err := node.CreatePageNode(ctx, page, builder.rw)
		if err != nil {
			return err
		}
		pageNode = nodeObj
	}

	parentNode.AddChild(pageNode)
	builder.nodeStack.Push(&stackContent{nodeObject: pageNode, objectId: pageId})
	return nil
}

// Query all the blocks of the page and add them to given node i.e. parentNode
func (builder *ExportTreeBuilder) queryAndAddPageChildren(ctx context.Context, parentNode *node.Node, pageId string) error {
	cursor := notionapi.Cursor("")
	for {
		blocks, cursor, err := builder.notionClient.GetPageBlocks(ctx, notionclient.PageID(pageId), cursor)

		if err != nil {
			return err
		}

		for _, block := range blocks {
			err = builder.addBlock(ctx, parentNode, block)
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
func (builder *ExportTreeBuilder) addDatabase(ctx context.Context, parentNode *node.Node, databaseId string) error {
	var databaseNode *node.Node

	if nodeObj, found := builder.databaseId2DatabaseNodeMap[databaseId]; found {
		databaseNode = nodeObj
	} else {
		database, err := builder.notionClient.GetDatabaseByID(ctx, notionclient.DatabaseID(databaseId))
		if err != nil {
			return err
		}

		nodeObj, err := node.CreateDatabaseNode(ctx, database, builder.rw)
		if err != nil {
			return err
		}
		databaseNode = nodeObj
	}

	parentNode.AddChild(databaseNode)
	builder.nodeStack.Push(&stackContent{nodeObject: databaseNode, objectId: databaseId})
	return nil
}

// Query all the pages of the given database and add them to the given node i.e
// parentNode
func (builder *ExportTreeBuilder) queryAndAddDatabaseChildren(ctx context.Context, parentNode *node.Node, databaseId string) error {
	cursor := notionapi.Cursor("")

	if pageIdList, found := builder.databaseId2PageListMap[databaseId]; found {
		for _, pageId := range pageIdList {
			err := builder.addPage(ctx, parentNode, pageId)

			if err != nil {
				return err
			}
		}

		return nil
	}

	for {
		pages, cursor, err := builder.notionClient.GetDatabasePages(ctx, notionclient.DatabaseID(databaseId), cursor)

		if err != nil {
			return err
		}

		for _, page := range pages {

			pageNode, err := node.CreatePageNode(ctx, &page, builder.rw)
			if err != nil {
				return err
			}
			parentNode.AddChild(pageNode)
			builder.nodeStack.Push(&stackContent{nodeObject: pageNode, objectId: page.ID.String()})
		}

		if cursor == "" {
			break
		}
	}

	return nil
}

// Identify the page for ChildPageBlock from page name and add it to tree
func (builder *ExportTreeBuilder) handleChildPageBlock(ctx context.Context, parentNode *node.Node, block notionapi.Block) error {
	childPage, ok := block.(notionapi.ChildPageBlock)
	if !ok {
		return errors.New("failed to cast block object to ChildPageBlock")
	}

	pageName := childPage.ChildPage.Title
	parentId := builder.getNonBlockTypeParentNotionObjectId(parentNode)
	cursor := notionapi.Cursor("")
	for {

		// There can be multiple pages with same name
		pages, cursor, err := builder.notionClient.GetPagesByName(ctx, notionclient.PageName(pageName), cursor)
		if err != nil {
			return err
		}

		pageFound := false
		var selectedPage *notionapi.Page
		for _, page := range pages {
			if page.Parent.Type == PARENT_TYPE_DATABASE && page.Parent.DatabaseID.String() == parentId {
				pageFound = true
				selectedPage = &page
				break
			} else if page.Parent.Type == PARENT_TYPE_PAGE && page.Parent.PageID.String() == parentId {
				pageFound = true
				selectedPage = &page
				break
			}
		}

		var pageNode *node.Node
		if pageFound {
			if nodeObj, found := builder.pageId2PageNodeMap[selectedPage.ID.String()]; found {
				pageNode = nodeObj
			} else {
				nodeObj, err := node.CreatePageNode(ctx, selectedPage, builder.rw)
				if err != nil {
					return err
				}
				pageNode = nodeObj
			}

			parentNode.AddChild(pageNode)
			builder.nodeStack.Push(&stackContent{nodeObject: pageNode, objectId: selectedPage.ID.String()})
			return nil
		}

		if cursor == "" {
			break
		}
	}

	return errors.New("page with name '" + childPage.ChildPage.Title + "' does not exist")
}

// Identify the database for ChildDatabaseBlock from database name and add it to tree
func (builder *ExportTreeBuilder) handleChildDatabaseBlock(ctx context.Context, parentNode *node.Node, block notionapi.Block) error {
	childDatabase, ok := block.(notionapi.ChildDatabaseBlock)
	if !ok {
		return errors.New("failed to cast block object to ChildPageBlock")
	}

	databaseName := childDatabase.ChildDatabase.Title
	parentId := builder.getNonBlockTypeParentNotionObjectId(parentNode)
	cursor := notionapi.Cursor("")
	for {

		// There can be multiple databases with same name
		databases, cursor, err := builder.notionClient.GetDatabasesByName(ctx, notionclient.DatabaseName(databaseName), cursor)
		if err != nil {
			return err
		}

		databaseFound := false
		var selectedDatabase *notionapi.Database
		for _, database := range databases {
			if database.Parent.Type == PARENT_TYPE_DATABASE && database.Parent.DatabaseID.String() == parentId {
				databaseFound = true
				selectedDatabase = &database
				break
			} else if database.Parent.Type == PARENT_TYPE_PAGE && database.Parent.PageID.String() == parentId {
				databaseFound = true
				selectedDatabase = &database
				break
			}
		}

		var databaseNode *node.Node
		if databaseFound {
			if nodeObj, found := builder.databaseId2DatabaseNodeMap[selectedDatabase.ID.String()]; found {
				databaseNode = nodeObj
			} else {
				nodeObj, err := node.CreateDatabaseNode(ctx, selectedDatabase, builder.rw)
				if err != nil {
					return err
				}
				databaseNode = nodeObj
			}

			parentNode.AddChild(databaseNode)
			builder.nodeStack.Push(&stackContent{nodeObject: databaseNode, objectId: selectedDatabase.ID.String()})
			return nil
		}

		if cursor == "" {
			break
		}
	}

	return errors.New("database with name '" + childDatabase.ChildDatabase.Title + "' does not exist")
}

// Create node object for given block and add it's children to created
// block node object
func (builder *ExportTreeBuilder) addBlock(ctx context.Context, parentNode *node.Node, block notionapi.Block) error {
	blockNode, err := node.CreateBlockNode(ctx, block, builder.rw)

	if err != nil {
		return err
	}
	parentNode.AddChild(blockNode)

	if !block.GetHasChildren() {
		return nil
	}

	if block.GetType() == CHILD_TYPE_PAGE {
		return builder.handleChildPageBlock(ctx, blockNode, block)
	}
	if block.GetType() == CHILD_TYPE_DATABASE {
		return builder.handleChildDatabaseBlock(ctx, blockNode, block)
	}

	builder.nodeStack.Push(&stackContent{nodeObject: blockNode, objectId: block.GetID().String()})
	return builder.queryAndAddBlockChildren(ctx, blockNode, string(block.GetID()))

}

// Query all the child blocks of the given block and add them to the given node
// i.e. parentNode
func (builder *ExportTreeBuilder) queryAndAddBlockChildren(ctx context.Context, parentNode *node.Node, blockId string) error {
	cursor := notionapi.Cursor("")
	for {
		blocks, cursor, err := builder.notionClient.GetChildBlocksOfBlock(ctx, notionclient.BlockID(blockId), cursor)

		if err != nil {
			return err
		}

		for _, block := range blocks {
			err = builder.addBlock(ctx, parentNode, block)
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

// This function will fetch all the pages. Pages which belong to workspace will
// be added to tree and stack and rest of them will be cached which will be
// later used while building the tree.
func (builder *ExportTreeBuilder) addWorkspacePages(ctx context.Context, parentNode *node.Node) error {
	cursor := notionapi.Cursor("")
	for {
		pages, cursor, err := builder.notionClient.GetAllPages(ctx, cursor)

		if err != nil {
			return err
		}

		for _, page := range pages {
			if builder.isParentWorkspace(&page.Parent) {
				pageNode, err := node.CreatePageNode(ctx, &page, builder.rw)
				if err != nil {
					return err
				}
				parentNode.AddChild(pageNode)
				object := &stackContent{
					nodeObject: pageNode,
					objectId:   page.ID.String(),
				}

				builder.nodeStack.Push(object)
				continue
			}

			// cache for later use
			pageNode, err := node.CreatePageNode(ctx, &page, builder.rw)
			if err != nil {
				return nil
			}

			builder.addDatabaseIdToPageMapping(&page)
			builder.pageId2PageNodeMap[string(page.ID)] = pageNode
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
func (builder *ExportTreeBuilder) addWorkspaceDatabases(ctx context.Context, parentNode *node.Node) error {
	cursor := notionapi.Cursor("")
	for {
		databases, cursor, err := builder.notionClient.GetAllDatabases(ctx, cursor)
		if err != nil {
			return err
		}

		for _, database := range databases {
			if builder.isParentWorkspace(&database.Parent) {
				databaseNode, err := node.CreateDatabaseNode(ctx, &database, builder.rw)
				if err != nil {
					return err
				}
				parentNode.AddChild(databaseNode)
				object := &stackContent{
					nodeObject: databaseNode,
					objectId:   database.ID.String(),
				}

				builder.nodeStack.Push(object)
				continue
			}

			// cache for later use
			databaseNode, err := node.CreateDatabaseNode(ctx, &database, builder.rw)
			if err != nil {
				return nil
			}

			builder.databaseId2DatabaseNodeMap[string(database.ID)] = databaseNode
		}

		if cursor == "" {
			break
		}
	}

	return nil
}

func (builder *ExportTreeBuilder) buildTreeUntilStackEmpty(ctx context.Context) error {
	for !builder.nodeStack.IsEmpty() {
		object, err := builder.nodeStack.Pop()
		if err == StackEmpty {
			break
		}

		if object.nodeObject.GetNodeType() == node.PAGE {
			err = builder.queryAndAddPageChildren(ctx, object.nodeObject, object.objectId)
		} else if object.nodeObject.GetNodeType() == node.DATABASE {
			err = builder.queryAndAddDatabaseChildren(ctx, object.nodeObject, object.objectId)
		} else if object.nodeObject.GetNodeType() == node.BLOCK {
			err = builder.queryAndAddBlockChildren(ctx, object.nodeObject, object.objectId)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// This function will build the tree for whole workspace depending on Databases
// and Pages the token has access to
func (builder *ExportTreeBuilder) buildTreeForWorkspace(ctx context.Context) error {
	rootNode := node.CreateRootNode()

	err := builder.addWorkspacePages(ctx, rootNode)
	if err != nil {
		return err
	}

	err = builder.addWorkspaceDatabases(ctx, rootNode)
	if err != nil {
		return err
	}

	err = builder.buildTreeUntilStackEmpty(ctx)
	if err != nil {
		return err
	}

	builder.rootNode = rootNode
	return nil
}

// Build the tree for the given config
func (builder *ExportTreeBuilder) BuildTree(ctx context.Context) error {
	if builder.rootNode != nil {
		return nil
	}

	builder.err = builder.buildTreeForWorkspace(ctx)
	return builder.err
}

// Get the root node of the tree
func (builder *ExportTreeBuilder) GetRootNode() (*node.Node, error) {
	return builder.rootNode, builder.err
}
