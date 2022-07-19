package node

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jomei/notionapi"
	"github.com/sawantshivaji1997/notionbackup/src/metadata"
	"github.com/sawantshivaji1997/notionbackup/src/rw"
)

type NodeID string
type NodeType string

func (id NodeID) String() string {
	return string(id)
}

const (
	UNKNOWN  NodeType = "UNKNOWN"
	ROOT     NodeType = "ROOT"
	PAGE     NodeType = "PAGE"
	DATABASE NodeType = "DATABASE"
	BLOCK    NodeType = "BLOCK"
)

type Node struct {
	id                NodeID
	nodeType          NodeType
	storageIdentifier rw.DataIdentifier
	notionObjectId    string

	// Using N-ary tree implementation
	// https://www.interviewbit.com/blog/n-ary-tree/
	sibling *Node
	child   *Node

	// Link to parent object
	parent *Node
}

// Helper function to create node with given NodeType
func createNode(id NodeID, nodeType NodeType,
	storageIdentifier rw.DataIdentifier, notionObjectId string) (*Node, error) {
	return &Node{
		id:                id,
		nodeType:          nodeType,
		sibling:           nil,
		child:             nil,
		storageIdentifier: storageIdentifier,
		notionObjectId:    notionObjectId,
	}, nil
}

// Create database node
func CreateDatabaseNode(ctx context.Context, database *notionapi.Database,
	rw rw.ReaderWriter) (*Node, error) {
	storageIdentifier, err := rw.WriteDatabase(ctx, database)
	if err != nil {
		return nil, err
	}

	return createNode(NodeID(uuid.New().String()), DATABASE, storageIdentifier,
		database.ID.String())
}

// Create page node
func CreatePageNode(ctx context.Context, page *notionapi.Page,
	rw rw.ReaderWriter) (*Node, error) {
	storageIdentifier, err := rw.WritePage(ctx, page)
	if err != nil {
		return nil, err
	}

	return createNode(NodeID(uuid.New().String()), PAGE, storageIdentifier,
		page.ID.String())
}

// Create block node
func CreateBlockNode(ctx context.Context, block notionapi.Block,
	rw rw.ReaderWriter) (*Node, error) {
	storageIdentifier, err := rw.WriteBlock(ctx, block)
	if err != nil {
		return nil, err
	}

	return createNode(NodeID(uuid.New().String()), BLOCK, storageIdentifier,
		block.GetID().String())
}

// Special node which will act as a root node for a tree
// Root node will always have Nil UUID i.e. 00000000-0000-0000-0000-000000000000
func CreateRootNode() *Node {
	return &Node{
		id:                NodeID(uuid.Nil.String()),
		notionObjectId:    "",
		nodeType:          ROOT,
		sibling:           nil,
		child:             nil,
		storageIdentifier: "",
		parent:            nil,
	}
}

func CreateNode(obj *metadata.NotionObject) (*Node, error) {
	var nodeType NodeType
	switch obj.Type {
	case metadata.NotionObjectType_ROOT:
		nodeType = ROOT
	case metadata.NotionObjectType_BLOCK:
		nodeType = BLOCK
	case metadata.NotionObjectType_DATABASE:
		nodeType = DATABASE
	case metadata.NotionObjectType_PAGE:
		nodeType = PAGE
	default:
		nodeType = UNKNOWN
	}

	if nodeType == UNKNOWN {
		return nil, fmt.Errorf("unknown notion object type: %s", nodeType)
	}

	if nodeType == ROOT {
		if obj.Uuid != uuid.Nil.String() {
			return nil, fmt.Errorf("not a valid root node. Uuid: %s", obj.Uuid)
		}

		if obj.NotionObjectId != "" {
			return nil, fmt.Errorf("not a valid root node. Notion object Id: %s",
				obj.NotionObjectId)
		}

		if obj.StorageIdentifier != "" {
			return nil, fmt.Errorf("not a valid root node. StorageIdentifier: %s",
				obj.StorageIdentifier)
		}
	}

	return &Node{
		id:                NodeID(obj.Uuid),
		nodeType:          nodeType,
		storageIdentifier: rw.DataIdentifier(obj.StorageIdentifier),
		notionObjectId:    obj.NotionObjectId,
		sibling:           nil,
		child:             nil,
		parent:            nil,
	}, nil
}

// Various getter function for getting various properties of Node object
func (nodeObj *Node) GetID() NodeID {
	return nodeObj.id
}

func (nodeObj *Node) GetNodeType() NodeType {
	return nodeObj.nodeType
}

func (nodeObj *Node) GetStorageIdentifier() rw.DataIdentifier {
	return nodeObj.storageIdentifier
}

func (nodeObj *Node) HasChildNode() bool {
	return nodeObj.child != nil
}

func (nodeObj *Node) HasSibling() bool {
	return nodeObj.sibling != nil
}

func (nodeObj *Node) GetChildNode() *Node {
	return nodeObj.child
}

func (nodeObj *Node) GetSiblingNode() *Node {
	return nodeObj.sibling
}

func (nodeObj *Node) GetNotionObjectId() string {
	return nodeObj.notionObjectId
}

func (nodeObj *Node) GetParentNode() *Node {
	return nodeObj.parent
}

// Adding a child to current node
func (nodeObj *Node) AddChild(childNode *Node) {
	childNode.parent = nodeObj
	if nodeObj.child == nil {
		nodeObj.child = childNode
	} else {
		tempNode := nodeObj.child

		for {
			if tempNode.sibling != nil {
				tempNode = tempNode.sibling
			} else {
				break
			}
		}

		tempNode.sibling = childNode
	}
}

// Delete a child with given node ID and return the deleted Node
func (nodeObj *Node) DeleteChild(id NodeID) *Node {
	if nodeObj.child == nil {
		return nil
	}

	// First Check if child object matches with give Node ID
	if nodeObj.child.id == id {
		tempNode := nodeObj.child
		if nodeObj.child.HasSibling() {
			nodeObj.child = nodeObj.child.GetSiblingNode()
		} else {
			nodeObj.child = nil
		}

		tempNode.sibling = nil
		tempNode.parent = nil
		return tempNode
	}

	if !nodeObj.child.HasSibling() {
		return nil
	}

	// Check all sibiling nodes
	head := nodeObj.child

	for {
		if head.HasSibling() && head.sibling.id == id {
			tempNode := head.sibling
			head.sibling = head.sibling.sibling

			tempNode.sibling = nil
			tempNode.parent = nil
			return tempNode
		}

		head = head.sibling
		if head == nil {
			break
		}
	}

	return nil
}
