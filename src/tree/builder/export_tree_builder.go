package builder

import (
	"context"
	"fmt"

	"github.com/jomei/notionapi"
	"github.com/rs/zerolog"
	"github.com/sawantshivaji1997/notionbackup/src/logging"
	"github.com/sawantshivaji1997/notionbackup/src/notionclient"
	"github.com/sawantshivaji1997/notionbackup/src/rw"
	"github.com/sawantshivaji1997/notionbackup/src/tree"
	"github.com/sawantshivaji1997/notionbackup/src/tree/node"
	"github.com/sawantshivaji1997/notionbackup/src/utils"
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
	objectId2NodeMap           map[string]*node.Node
}

func GetExportTreebuilder(ctx context.Context,
	notionClient notionclient.NotionClient, rw rw.ReaderWriter,
	request *TreeBuilderRequest) TreeBuilder {
	return &ExportTreeBuilder{
		notionClient:               notionClient,
		rw:                         rw,
		err:                        fmt.Errorf("tree was never built"),
		rootNode:                   nil,
		pageId2PageNodeMap:         make(map[string]*node.Node),
		databaseId2DatabaseNodeMap: make(map[string]*node.Node),
		databaseId2PageListMap:     make(map[string][]string),
		nodeStack:                  make(stack, 0),
		request:                    request,
		objectId2NodeMap:           make(map[string]*node.Node),
	}
}

// Check if Parent type is workspace
func (builderObj *ExportTreeBuilder) isParentWorkspace(
	parent *notionapi.Parent) bool {
	return parent.Type == notionapi.ParentTypeWorkspace
}

// Helper function to add database ID to Page Id list mapping
func (builderObj *ExportTreeBuilder) addDatabaseIdToPageMapping(
	page *notionapi.Page) {
	if page.Parent.Type != notionapi.ParentTypeDatabaseID {
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

// Checks if notion object with ID is already added to tree
func (builderObj *ExportTreeBuilder) getNode(
	notionObjectID string) *node.Node {
	nodeObj := builderObj.objectId2NodeMap[notionObjectID]
	return nodeObj
}

// This function handles the scenario where user has provided multiple page and
// database IDs to backup and some pages/databases might be child of another
// page or database. In such cases, we don't want to create two copies same page
// or database node (one node child of root node while other node child of
// actual node to which it belongs)
// Example: Page P1 has child Page P2, Page P2 has child Page P3. Now user
// provides UUID of Page P1 and P3 to backup. In such scenarios, we don't want
// to make Page P3 child of root node as it will be automatically made child of
// Page P2 while backing up whole Page P1
// This function handles such scenario where it modifies the tree accordingly
// adds the node as a child appropriate node
func (builderObj *ExportTreeBuilder) restructureTree(ctx context.Context,
	restructureNode *node.Node, parentNode *node.Node) error {
	objectTypeUuid := logging.PageUUID
	if restructureNode.GetNodeType() == node.DATABASE {
		objectTypeUuid = logging.DatabaseUUID
	}

	log := zerolog.Ctx(ctx).With().Str(objectTypeUuid,
		restructureNode.GetNotionObjectId()).Logger()
	log.Debug().Msgf("Restructuring the %s Node", restructureNode.GetNodeType())

	if restructureNode.GetParentNode().GetNodeType() == node.ROOT {
		deletedNode := restructureNode.GetParentNode().DeleteChild(
			restructureNode.GetID())

		if deletedNode == nil {
			errMsg := fmt.Sprintf("%s node with UUID '%s' does not exist",
				restructureNode.GetNodeType(), restructureNode.GetNotionObjectId())
			return fmt.Errorf(errMsg)
		}

		if deletedNode.GetID() != restructureNode.GetID() {
			errMsg := fmt.Sprintf("Invalid node deleted: Node to Delete: (%s, %s), "+
				"Deleted Node: (%s, %s)",
				restructureNode.GetNodeType(), restructureNode.GetNotionObjectId(),
				deletedNode.GetNodeType(), deletedNode.GetNotionObjectId())
			return fmt.Errorf(errMsg)
		}

		parentNode.AddChild(restructureNode)
	}

	return nil
}

// Create node object for given page and add it's children to created page node
// object
func (builderObj *ExportTreeBuilder) addPage(ctx context.Context,
	parentNode *node.Node, pageId string) error {
	log := zerolog.Ctx(ctx).With().Str(logging.PageUUID, pageId).Logger()
	var pageNode *node.Node

	if nodeObj, found := builderObj.pageId2PageNodeMap[pageId]; found {
		pageNode = nodeObj
		delete(builderObj.pageId2PageNodeMap, pageId)
	} else if foundNode := builderObj.getNode(pageId); foundNode != nil {
		return builderObj.restructureTree(ctx, foundNode, parentNode)
	} else {
		log.Debug().Msg("Fetching Page")
		page, err := builderObj.notionClient.GetPageByID(ctx,
			notionclient.PageID(pageId))
		if err != nil {
			log.Error().Err(err).Msg(logging.PageFetchErr)
			return err
		}
		nodeObj, err := node.CreatePageNode(ctx, page, builderObj.rw)
		if err != nil {
			log.Error().Err(err).Msg(logging.PageNodeCreateErr)
			return err
		}

		builderObj.objectId2NodeMap[nodeObj.GetNotionObjectId()] = nodeObj
		pageNode = nodeObj
	}

	parentNode.AddChild(pageNode)
	builderObj.nodeStack.Push(pageNode)
	return nil
}

// Query all the blocks of the page and add them to given node i.e. parentNode
func (builderObj *ExportTreeBuilder) queryAndAddPageChildren(
	ctx context.Context, parentNode *node.Node, pageId string) error {
	log := zerolog.Ctx(ctx).With().Str(logging.PageUUID, pageId).Logger()
	log.Debug().Msg("Fetching Page blocks")

	cursor := notionapi.Cursor("")
	for {
		var blocks []notionapi.Block
		var err error
		blocks, cursor, err = builderObj.notionClient.GetPageBlocks(
			ctx, notionclient.PageID(pageId), cursor)

		if err != nil {
			log.Error().Err(err).Msg(logging.PageBlocksFetchErr)
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
	log := zerolog.Ctx(ctx).With().Str(logging.DatabaseUUID, databaseId).Logger()
	var databaseNode *node.Node

	if nodeObj, found := builderObj.
		databaseId2DatabaseNodeMap[databaseId]; found {
		databaseNode = nodeObj
		delete(builderObj.databaseId2DatabaseNodeMap, databaseId)
	} else if foundNode := builderObj.getNode(databaseId); foundNode != nil {
		return builderObj.restructureTree(ctx, foundNode, parentNode)
	} else {
		log.Debug().Msg("Fetching Database")
		database, err := builderObj.notionClient.GetDatabaseByID(ctx,
			notionclient.DatabaseID(databaseId))

		if err != nil {
			log.Error().Err(err).Msg(logging.DatabaseFetchErr)
			return err
		}

		nodeObj, err := node.CreateDatabaseNode(ctx, database, builderObj.rw)
		if err != nil {
			log.Error().Err(err).Msg(logging.DatabaseNodeCreateErr)
			return err
		}

		builderObj.objectId2NodeMap[nodeObj.GetNotionObjectId()] = nodeObj
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
	log := zerolog.Ctx(ctx).With().Str(logging.DatabaseUUID, databaseId).Logger()

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

	log.Debug().Msg("Fetching Database pages")
	cursor := notionapi.Cursor("")
	for {
		var pages []notionapi.Page
		var err error
		pages, cursor, err = builderObj.notionClient.GetDatabasePages(ctx,
			notionclient.DatabaseID(databaseId), cursor)

		if err != nil {
			log.Error().Err(err).Msg(logging.DatabasePagesFetchErr)
			return err
		}

		for _, page := range pages {

			if foundNode := builderObj.getNode(page.ID.String()); foundNode != nil {
				err = builderObj.restructureTree(ctx, foundNode, parentNode)
				if err != nil {
					log.Error().Err(err).Str(logging.PageUUID, page.ID.String()).
						Msg("Failed to restructure the tree for database page node")
					return err
				}
			}

			pageNode, err := node.CreatePageNode(ctx, &page, builderObj.rw)
			if err != nil {
				log.Error().Err(err).Str(logging.PageUUID, page.ID.String()).
					Msg(logging.PageNodeCreateErr)
				return err
			}

			builderObj.objectId2NodeMap[pageNode.GetNotionObjectId()] = pageNode
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
	log := zerolog.Ctx(ctx)
	blockNode, err := node.CreateBlockNode(ctx, block, builderObj.rw)

	if err != nil {
		log.Error().Err(err).Str(logging.BlockUUID, block.GetID().String()).
			Msg(logging.BlockNodeCreateErr)
		return err
	}

	parentNode.AddChild(blockNode)

	if block.GetType() == notionapi.BlockTypeChildDatabase {
		return builderObj.addDatabase(ctx, blockNode, block.GetID().String())
	}

	if block.GetType() == notionapi.BlockTypeChildPage {
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
	log := zerolog.Ctx(ctx).With().Str(logging.BlockUUID, blockId).Logger()
	log.Debug().Msg("Fetching child blocks")

	cursor := notionapi.Cursor("")
	for {
		var blocks []notionapi.Block
		var err error
		blocks, cursor, err = builderObj.notionClient.GetChildBlocksOfBlock(
			ctx, notionclient.BlockID(blockId), cursor)

		if err != nil {
			log.Error().Err(err).Msg(logging.ChildBlockFetchErr)
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
	log := zerolog.Ctx(ctx)
	log.Debug().Msg("Fetching all pages from the workspace")
	cursor := notionapi.Cursor("")
	for {
		var pages []notionapi.Page
		var err error
		pages, cursor, err = builderObj.notionClient.GetAllPages(ctx, cursor)

		if err != nil {
			log.Error().Err(err).Msg(logging.PageFetchErr)
			return err
		}

		for _, page := range pages {
			if builderObj.isParentWorkspace(&page.Parent) {
				pageNode, err := node.CreatePageNode(ctx, &page, builderObj.rw)
				if err != nil {
					log.Error().Err(err).Str(logging.PageUUID, page.ID.String()).
						Msg(logging.PageNodeCreateErr)
					return err
				}

				parentNode.AddChild(pageNode)
				builderObj.nodeStack.Push(pageNode)
				continue
			}

			// cache for later use
			pageNode, err := node.CreatePageNode(ctx, &page, builderObj.rw)
			if err != nil {
				log.Error().Err(err).Str(logging.PageUUID, page.ID.String()).
					Msg(logging.PageNodeCreateErr)
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
	log := zerolog.Ctx(ctx)
	log.Debug().Msg("Fetching all databases from the workspace")
	cursor := notionapi.Cursor("")
	for {
		var databases []notionapi.Database
		var err error
		databases, cursor, err = builderObj.notionClient.GetAllDatabases(
			ctx, cursor)
		if err != nil {
			log.Error().Err(err).Msg(logging.DatabaseFetchErr)
			return err
		}

		for _, database := range databases {
			if builderObj.isParentWorkspace(&database.Parent) {
				databaseNode, err := node.CreateDatabaseNode(
					ctx, &database, builderObj.rw)
				if err != nil {
					log.Error().Err(err).Str(logging.DatabaseUUID, database.ID.String()).
						Msg(logging.DatabaseNodeCreateErr)
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
				log.Error().Err(err).Str(logging.DatabaseUUID, database.ID.String()).
					Msg(logging.DatabaseNodeCreateErr)
				return err
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
		if err == errStackEmpty {
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

	builderObj.request.PageIdList = utils.
		GetUniqueValues(builderObj.request.PageIdList)
	builderObj.request.DatabaseIdList = utils.
		GetUniqueValues(builderObj.request.DatabaseIdList)

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
func (builderObj *ExportTreeBuilder) BuildTree(ctx context.Context) (*tree.Tree,
	error) {
	log := zerolog.Ctx(ctx)

	if builderObj.rootNode != nil {
		return &tree.Tree{
			RootNode: builderObj.rootNode,
		}, nil
	}

	// If no PageIDs and DatabaseIDs provided, build tree with all pages and
	// databases from workspace which user has access
	if len(builderObj.request.DatabaseIdList) == 0 &&
		len(builderObj.request.PageIdList) == 0 {
		log.Debug().Msgf("Building tree for whole workspace")
		builderObj.err = builderObj.buildTreeForWorkspace(ctx)
	} else {
		log.Debug().Msgf("Building tree for given notion objects UUID")
		builderObj.err = builderObj.buildTreeForGivenObjectIds(ctx)
	}

	if builderObj.err != nil {
		log.Error().Err(builderObj.err).Msg(
			"Failed to build the export tree. Cleaning up...")

		err := builderObj.rw.CleanUp(ctx)
		if err != nil {
			log.Warn().Err(err).Msg(
				"Failed to cleanup the exported data. Manual cleanup may be required")
		} else {
			log.Info().Msg("Cleanup successful")
		}

		return nil, builderObj.err
	}

	log.Debug().Msg("Successfully built export tree")
	return &tree.Tree{
		RootNode: builderObj.rootNode,
	}, nil
}
