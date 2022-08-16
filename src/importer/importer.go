package importer

import (
	"container/list"
	"context"
	"fmt"

	"github.com/jomei/notionapi"
	"github.com/sawantshivaji1997/notionbackup/src/notionclient"
	"github.com/sawantshivaji1997/notionbackup/src/rw"
	"github.com/sawantshivaji1997/notionbackup/src/tree"
	"github.com/sawantshivaji1997/notionbackup/src/tree/iterator"
	"github.com/sawantshivaji1997/notionbackup/src/tree/node"
)

const (
	PARENT_TYPE_WORKSPACE = notionapi.ParentType("workspace")
	PARENT_TYPE_DATABASE  = notionapi.ParentType("database_id")
	PARENT_TYPE_PAGE      = notionapi.ParentType("page_id")
	CHILD_TYPE_PAGE       = notionapi.BlockType("child_page")
	CHILD_TYPE_DATABASE   = notionapi.BlockType("child_database")
)

type objectUuidMapping struct {
	pageMap     map[notionapi.PageID]notionapi.PageID
	databaseMap map[notionapi.DatabaseID]notionapi.DatabaseID
	blockMap    map[notionapi.BlockID]notionapi.BlockID
}

func (o *objectUuidMapping) insertPageUuid(oldUuid,
	newUuid notionapi.ObjectID) {
	o.pageMap[notionapi.PageID(oldUuid)] = notionapi.PageID(newUuid)
}

func (o *objectUuidMapping) getPageUuid(
	oldUuid notionapi.PageID) (notionapi.PageID, error) {
	newUuid, found := o.pageMap[oldUuid]
	if !found {
		return "", fmt.Errorf("new uuid for page %s does not exist", oldUuid)
	}

	return newUuid, nil
}

func (o *objectUuidMapping) insertDatabaseUuid(oldUuid,
	newUuid notionapi.ObjectID) {
	o.databaseMap[notionapi.DatabaseID(oldUuid)] = notionapi.DatabaseID(newUuid)
}

func (o *objectUuidMapping) getDatabaseUuid(
	oldUuid notionapi.DatabaseID) (notionapi.DatabaseID, error) {
	newUuid, found := o.databaseMap[oldUuid]
	if !found {
		return "", fmt.Errorf("new uuid for database %s does not exist", oldUuid)
	}

	return newUuid, nil
}

func (o *objectUuidMapping) insertBlockUuid(oldUuid,
	newUuid notionapi.ObjectID) {
	o.blockMap[notionapi.BlockID(oldUuid)] = notionapi.BlockID(newUuid)
}

func (o *objectUuidMapping) getBlockUuid(
	oldUuid notionapi.BlockID) (notionapi.BlockID, error) {
	newUuid, found := o.blockMap[oldUuid]
	if !found {
		return "", fmt.Errorf("new uuid for block %s does not exist", oldUuid)
	}

	return newUuid, nil
}

type Importer struct {
	rwClient       rw.ReaderWriter
	notionClient   notionclient.NotionClient
	treeObj        *tree.Tree
	objUuidMapping *objectUuidMapping
	nodeQueue      *list.List
}

func GetImporter(rwClient rw.ReaderWriter,
	notionClient notionclient.NotionClient, treeObj *tree.Tree) *Importer {
	return &Importer{
		rwClient:     rwClient,
		notionClient: notionClient,
		treeObj:      treeObj,
		nodeQueue:    list.New(),
		objUuidMapping: &objectUuidMapping{
			pageMap:     make(map[notionapi.PageID]notionapi.PageID),
			databaseMap: make(map[notionapi.DatabaseID]notionapi.DatabaseID),
			blockMap:    make(map[notionapi.BlockID]notionapi.BlockID),
		},
	}
}

// This function creates and returns Parent object
func (c *Importer) getParentObject(nodeObj *node.Node,
	oldParent *notionapi.Parent) (*notionapi.Parent, error) {

	if nodeObj.GetParentNode().GetNodeType() == node.ROOT {
		return &notionapi.Parent{
			Type: PARENT_TYPE_WORKSPACE,
		}, nil
	}

	newParent := &notionapi.Parent{
		Type: oldParent.Type,
	}

	if oldParent.Type == PARENT_TYPE_DATABASE {
		newUuid, err := c.objUuidMapping.getDatabaseUuid(oldParent.DatabaseID)

		if err != nil {
			return nil, err
		}

		newParent.DatabaseID = newUuid
	} else if oldParent.Type == PARENT_TYPE_PAGE {
		newUuid, err := c.objUuidMapping.getPageUuid(oldParent.PageID)

		if err != nil {
			return nil, err
		}

		newParent.PageID = newUuid
	} else {
		return nil, fmt.Errorf("unknown parent object type: %s", oldParent.Type)
	}

	return newParent, nil
}

// This function processes node object, creates Page request and uploads
// it to Notion
func (c *Importer) uploadPage(ctx context.Context, nodeObj *node.Node) error {
	page, err := c.rwClient.ReadPage(ctx, nodeObj.GetStorageIdentifier())
	if err != nil {
		return err
	}

	parent, err := c.getParentObject(nodeObj, &page.Parent)
	if err != nil {
		return err
	}

	req := &notionapi.PageCreateRequest{
		Parent:     *parent,
		Properties: page.Properties,
		Children:   make([]notionapi.Block, 0),
		Icon:       page.Icon,
		Cover:      page.Cover,
	}

	createdPage, err := c.notionClient.CreatePage(ctx, req)
	if err != nil {
		return err
	}

	c.objUuidMapping.insertPageUuid(page.ID, createdPage.ID)
	c.nodeQueue.PushBack(nodeObj)
	return nil
}

// This function to processes node object, creates Database request and uploads
// it to Notion
func (c *Importer) uploadDatabase(ctx context.Context,
	nodeObj *node.Node) error {
	database, err := c.rwClient.ReadDatabase(ctx, nodeObj.GetStorageIdentifier())
	if err != nil {
		return err
	}

	parent, err := c.getParentObject(nodeObj, &database.Parent)
	if err != nil {
		return err
	}

	req := &notionapi.DatabaseCreateRequest{
		Parent:     *parent,
		Title:      database.Title,
		Properties: database.Properties,
	}

	createdDatabase, err := c.notionClient.CreateDatabase(ctx, req)
	if err != nil {
		return err
	}

	c.objUuidMapping.insertDatabaseUuid(database.ID, createdDatabase.ID)
	c.nodeQueue.PushBack(nodeObj)
	return nil
}

// This function will upload the blocks to given block/page
func (c *Importer) uploadBlocks(ctx context.Context, parentUuid string,
	blocks notionapi.Blocks) error {
	if len(blocks) == 0 {
		return nil
	}

	req := &notionapi.AppendBlockChildrenRequest{
		Children: blocks,
	}

	rsp, err := c.notionClient.AppendBlocksToBlock(
		ctx, notionclient.BlockID(parentUuid), req)

	if err != nil {
		return err
	}

	// Ideally, length should always be equal
	if len(req.Children) != len(rsp.Results) {
		return fmt.Errorf("number of blocks in request does not match number of " +
			"blocks in response")
	}

	for i := range blocks {
		c.objUuidMapping.insertBlockUuid(notionapi.ObjectID(blocks[i].GetID()),
			notionapi.ObjectID(rsp.Results[i].GetID()))
	}

	return nil
}

// This function will iterate all block nodes of given node which can be page
//node or block node and upload it to Notion
func (c *Importer) processChildrenNodes(ctx context.Context, parentUuid string,
	nodeObj *node.Node) error {
	blocksIter := iterator.GetChildIterator(nodeObj)

	blockList := notionapi.Blocks{}

	for {
		childObj, err := blocksIter.Next()

		if err == iterator.ErrDone {
			break
		}

		block, err := c.rwClient.ReadBlock(ctx, childObj.GetStorageIdentifier())
		if err != nil {
			return err
		}

		if block.GetType() == CHILD_TYPE_PAGE ||
			block.GetType() == CHILD_TYPE_DATABASE {
			err = c.uploadBlocks(ctx, parentUuid, blockList)
			if err != nil {
				return err
			}

			// No need of creating separate block for Database or Page object. Once,
			// the Database or Page gets uploaded, the block would be automatically
			// created
			if block.GetType() == CHILD_TYPE_PAGE {
				err = c.uploadPage(ctx, childObj.GetChildNode())
			} else {
				err = c.uploadDatabase(ctx, childObj.GetChildNode())
			}

			if err != nil {
				return err
			}

			blockList = notionapi.Blocks{}
		} else {
			if childObj.HasChildNode() {
				c.nodeQueue.PushBack(childObj)
			}

			blockList = append(blockList, block)
		}
	}

	return c.uploadBlocks(ctx, parentUuid, blockList)
}

// This function will iterate all child nodes of Page node and upload it to
// Notion
func (c *Importer) processPageNode(ctx context.Context,
	nodeObj *node.Node) error {
	newPageUuid, err := c.objUuidMapping.getPageUuid(
		notionapi.PageID(nodeObj.GetNotionObjectId()))

	if err != nil {
		return err
	}

	return c.processChildrenNodes(ctx, newPageUuid.String(), nodeObj)
}

// This function will iterate all child nodes of Database node and upload it to
// Notion
func (c *Importer) processDatabaseNode(ctx context.Context,
	nodeObj *node.Node) error {
	childIter := iterator.GetChildIterator(nodeObj)
	for {
		childObj, err := childIter.Next()
		if err == iterator.ErrDone {
			break
		}

		// Ideally, database node should only have page as child nodes
		err = c.uploadPage(ctx, childObj)
		if err != nil {
			return err
		}
	}

	return nil
}

// This function will iterate all child nodes of block node and upload it to
// Notion
func (c *Importer) processBlockNode(ctx context.Context,
	nodeObj *node.Node) error {
	newBlockUuid, err := c.objUuidMapping.getBlockUuid(
		notionapi.BlockID(nodeObj.GetNotionObjectId()))

	if err != nil {
		return err
	}

	return c.processChildrenNodes(ctx, newBlockUuid.String(), nodeObj)
}

// This function will iterate all child nodes of root node and upload it to
// Notion
func (c *Importer) processRootNode(ctx context.Context,
	nodeObj *node.Node) error {
	childIter := iterator.GetChildIterator(nodeObj)
	for {
		childObj, err := childIter.Next()
		if err == iterator.ErrDone {
			break
		}

		// Root node should always have Page and Database as child nodes and no
		// block nodes
		if childObj.GetNodeType() == node.PAGE {
			err := c.uploadPage(ctx, childObj)
			if err != nil {
				return err
			}
		} else if childObj.GetNodeType() == node.DATABASE {
			err := c.uploadDatabase(ctx, childObj)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Check node object type and process them accordingly
func (c *Importer) processNodeObject(ctx context.Context,
	nodeObj *node.Node) error {
	if nodeObj.GetNodeType() == node.ROOT {
		return c.processRootNode(ctx, nodeObj)
	} else if nodeObj.GetNodeType() == node.DATABASE {
		return c.processDatabaseNode(ctx, nodeObj)
	} else if nodeObj.GetNodeType() == node.PAGE {
		return c.processPageNode(ctx, nodeObj)
	} else if nodeObj.GetNodeType() == node.BLOCK {
		return c.processBlockNode(ctx, nodeObj)
	}

	return fmt.Errorf("unknown node object type: %s", nodeObj.GetNodeType())
}

// Import all objects from tree
func (c *Importer) ImportObjects(ctx context.Context) error {
	c.nodeQueue.PushBack(c.treeObj.RootNode)
	for {
		if c.nodeQueue.Len() == 0 {
			break
		}

		front := c.nodeQueue.Front()
		currNode, ok := front.Value.(*node.Node)
		if !ok {
			return fmt.Errorf("failed to parse node object")
		}

		err := c.processNodeObject(ctx, currNode)
		if err != nil {
			return err
		}

		c.nodeQueue.Remove(front)
	}

	return nil
}
