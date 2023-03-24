package importer

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"

	"github.com/jomei/notionapi"
	"github.com/rs/zerolog"
	"github.com/shivaji17/notionbackup/src/notionclient"
	"github.com/shivaji17/notionbackup/src/rw"
	"github.com/shivaji17/notionbackup/src/tree"
	"github.com/shivaji17/notionbackup/src/tree/iterator"
	"github.com/shivaji17/notionbackup/src/tree/node"
	"github.com/shivaji17/notionbackup/src/utils"
)

var FIELDS_TO_CLEAR = []string{
	"id",
	"created_time",
	"last_edited_time",
	"created_by",
	"last_edited_by",
	"has_children",
}

type Importer struct {
	rwClient          rw.ReaderWriter
	notionClient      notionclient.NotionClient
	treeObj           *tree.Tree
	objUuidMapping    *objectUuidMapping
	nodeQueue         *list.List
	restoreToPageUUID string
}

func GetImporter(rwClient rw.ReaderWriter,
	notionClient notionclient.NotionClient, restoreToPageUUID string,
	treeObj *tree.Tree) *Importer {
	return &Importer{
		rwClient:          rwClient,
		notionClient:      notionClient,
		treeObj:           treeObj,
		nodeQueue:         list.New(),
		restoreToPageUUID: restoreToPageUUID,
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
			Type:   notionapi.ParentTypePageID,
			PageID: notionapi.PageID(c.restoreToPageUUID),
		}, nil
	}

	newParent := &notionapi.Parent{
		Type: oldParent.Type,
	}

	if oldParent.Type == notionapi.ParentTypeDatabaseID {
		newUuid, err := c.objUuidMapping.getDatabaseUuid(oldParent.DatabaseID)

		if err != nil {
			return nil, err
		}

		newParent.DatabaseID = newUuid
	} else if oldParent.Type == notionapi.ParentTypePageID {
		newUuid, err := c.objUuidMapping.getPageUuid(oldParent.PageID)

		if err != nil {
			return nil, err
		}

		newParent.PageID = newUuid
	} else if oldParent.Type == notionapi.ParentTypeBlockID {
		newUuid, err := c.objUuidMapping.getBlockUuid(oldParent.BlockID)

		if err != nil {
			return nil, err
		}
		newParent.BlockID = newUuid

	} else {
		return nil, fmt.Errorf("unknown parent object type: %s", oldParent.Type)
	}

	return newParent, nil
}

// This function processes node object, creates Page request and uploads
// it to Notion
func (c *Importer) uploadPage(ctx context.Context, nodeObj *node.Node) error {
	log := zerolog.Ctx(ctx)
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

	log.Debug().Msgf("Uploading Page %s...", page.ID)
	createdPage, err := c.notionClient.CreatePage(ctx, req)
	if err != nil {
		b, _ := json.Marshal(req)
		log.Err(err).Msgf("Page Create Request: %s", b)
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
	log := zerolog.Ctx(ctx)
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

	log.Debug().Msgf("Uploading Database %s...", database.ID)
	createdDatabase, err := c.notionClient.CreateDatabase(ctx, req)
	if err != nil {
		b, _ := json.Marshal(req)
		log.Err(err).Msgf("Database Create Request: %s", b)
		return err
	}

	c.objUuidMapping.insertDatabaseUuid(database.ID, createdDatabase.ID)
	c.nodeQueue.PushBack(nodeObj)
	return nil
}

func (c *Importer) createMappingForColumnList(ctx context.Context,
	oldBlock notionapi.Block, newBlock notionapi.Block) error {
	oldColumnsList, ok := oldBlock.(*notionapi.ColumnListBlock)
	if !ok {
		return fmt.Errorf("failed to cast block object to columnlist object")
	}

	rsp, _, err := c.notionClient.GetChildBlocksOfBlock(ctx,
		notionclient.BlockID(newBlock.GetID()), notionapi.Cursor(""))
	if err != nil {
		return err
	}

	if len(oldColumnsList.ColumnList.Children) != len(rsp) {
		return fmt.Errorf("number of original columns is not equal to number " +
			" of created columns")
	}

	for i := range oldColumnsList.ColumnList.Children {
		c.objUuidMapping.insertBlockUuid(
			notionapi.ObjectID(oldColumnsList.ColumnList.Children[i].GetID()),
			notionapi.ObjectID(rsp[i].GetID()))
	}

	return nil
}

// This function will upload the blocks to given block/page
func (c *Importer) uploadBlocks(ctx context.Context, parentUuid string,
	blocks notionapi.Blocks) error {
	log := zerolog.Ctx(ctx).With().Str("Parent UUID", parentUuid).Logger()
	if len(blocks) == 0 {
		return nil
	}

	req := &notionapi.AppendBlockChildrenRequest{
		Children: blocks,
	}

	var oldBlocks notionapi.Blocks
	for i := range blocks {
		oldBlocks = append(oldBlocks, copyBlock(blocks[i]))
		c.clear(&blocks[i])
	}

	log.Debug().Msg("Appending blocks...")
	rsp, err := c.notionClient.AppendBlocksToBlock(
		ctx, notionclient.BlockID(parentUuid), req)

	if err != nil {
		b, _ := json.Marshal(req)
		log.Err(err).Msgf("Block Append Request: %s", b)
		return err
	}

	// Ideally, length should always be equal
	if len(req.Children) != len(rsp.Results) {
		return fmt.Errorf("number of blocks in request does not match number of " +
			"blocks in response")
	}

	for i := range oldBlocks {
		c.objUuidMapping.insertBlockUuid(notionapi.ObjectID(oldBlocks[i].GetID()),
			notionapi.ObjectID(rsp.Results[i].GetID()))

		if oldBlocks[i].GetType() == notionapi.BlockTypeColumnList {
			err = c.createMappingForColumnList(ctx, oldBlocks[i], rsp.Results[i])
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Clear basic fields
func (c *Importer) clear(block *notionapi.Block) error {
	dataBytes, err := json.Marshal(block)
	if err != nil {
		return err
	}

	var response map[string]interface{}
	err = json.Unmarshal(dataBytes, &response)
	if err != nil {
		return err
	}

	for _, field := range FIELDS_TO_CLEAR {
		delete(response, field)
	}

	*block, err = utils.DecodeBlockObject(response)
	return err
}

// Handling for table object
func (c *Importer) handleTableObject(ctx context.Context, nodeObj *node.Node,
	block notionapi.Block) (notionapi.Block, error) {
	table, ok := block.(*notionapi.TableBlock)
	if !ok {
		return nil, fmt.Errorf("failed to cast block object to table object")
	}

	iter := iterator.GetChildIterator(nodeObj)
	for {
		childObj, err := iter.Next()
		if err == iterator.ErrDone {
			break
		}

		tableRow, err := c.rwClient.ReadBlock(ctx, childObj.GetStorageIdentifier())
		if err != nil {
			return nil, err
		}

		table.Table.Children = append(table.Table.Children, tableRow)
	}

	return table, nil
}

// Handling for columnlist object
func (c *Importer) handleColumnListObject(ctx context.Context,
	nodeObj *node.Node, block notionapi.Block) (notionapi.Block, error) {
	columnList, ok := block.(*notionapi.ColumnListBlock)
	if !ok {
		return nil, fmt.Errorf("failed to cast block object to columnlist object")
	}

	iter := iterator.GetChildIterator(nodeObj)
	for {
		childObj, err := iter.Next()
		if err == iterator.ErrDone {
			break
		}

		column, err := c.rwClient.ReadBlock(ctx, childObj.GetStorageIdentifier())
		if err != nil {
			return nil, err
		}

		columnList.ColumnList.Children =
			append(columnList.ColumnList.Children, column)

		if childObj.HasChildNode() {
			c.nodeQueue.PushBack(childObj)
		}
	}

	return columnList, nil
}

// Handling for block objects
func (c *Importer) handleBlockObject(ctx context.Context, nodeObj *node.Node,
	block notionapi.Block) (notionapi.Block, error) {

	if block.GetType() == notionapi.BlockTypeTableBlock {
		return c.handleTableObject(ctx, nodeObj, block)
	} else if block.GetType() == notionapi.BlockTypeColumnList {
		return c.handleColumnListObject(ctx, nodeObj, block)
	}

	if nodeObj.HasChildNode() {
		c.nodeQueue.PushBack(nodeObj)
	}

	return block, nil
}

// This function will iterate all block nodes of given node which can be page
// node or block node and upload it to Notion
func (c *Importer) processChildrenNodes(ctx context.Context, parentUuid string,
	nodeObj *node.Node) error {
	log := zerolog.Ctx(ctx)
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

		if block.GetType() == notionapi.BlockTypeUnsupported {
			log.Warn().Msgf("Unsupported block type encountered. Skipping restore "+
				"for block: %s", block.GetID())
			continue
		}

		if block.GetType() == notionapi.BlockTypeChildPage ||
			block.GetType() == notionapi.BlockTypeChildDatabase {
			err = c.uploadBlocks(ctx, parentUuid, blockList)
			if err != nil {
				return err
			}

			// No need of creating separate block for Database or Page object. Once,
			// the Database or Page gets uploaded, the block would be automatically
			// created
			if block.GetType() == notionapi.BlockTypeChildPage {
				err = c.uploadPage(ctx, childObj.GetChildNode())
			} else {
				err = c.uploadDatabase(ctx, childObj.GetChildNode())
			}

			if err != nil {
				return err
			}

			blockList = notionapi.Blocks{}
		} else {
			block, err = c.handleBlockObject(ctx, childObj, block)
			if err != nil {
				return err
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
	log := zerolog.Ctx(ctx)
	log.Debug().Msgf("Processing %s node", nodeObj.GetNodeType())

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
