package node

import (
	"context"

	"github.com/google/uuid"
	"github.com/jomei/notionapi"
	"github.com/sawantshivaji1997/notionbackup/src/rw"
)

type NodeID string
type NodeType int
type FilePath string

const (
	UNKNOWN  NodeType = 0
	ROOT              = 1
	PAGE              = 2
	DATABASE          = 3
	BLOCK             = 4
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
func createNode(id NodeID, nodeType NodeType, storageIdentifier rw.DataIdentifier, notionObjectId string) (*Node, error) {
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
func CreateDatabaseNode(ctx context.Context, database *notionapi.Database, rw rw.ReaderWriter) (*Node, error) {
	storageIdentifier, err := rw.WriteDatabase(ctx, database)
	if err != nil {
		return nil, err
	}

	return createNode(NodeID(uuid.New().String()), DATABASE, storageIdentifier, database.ID.String())
}

// Create page node
func CreatePageNode(ctx context.Context, page *notionapi.Page, rw rw.ReaderWriter) (*Node, error) {
	storageIdentifier, err := rw.WritePage(ctx, page)
	if err != nil {
		return nil, err
	}

	return createNode(NodeID(uuid.New().String()), PAGE, storageIdentifier, page.ID.String())
}

// Create block node
func CreateBlockNode(ctx context.Context, block notionapi.Block, rw rw.ReaderWriter) (*Node, error) {
	storageIdentifier, err := rw.WriteBlock(ctx, block)
	if err != nil {
		return nil, err
	}

	return createNode(NodeID(uuid.New().String()), BLOCK, storageIdentifier, block.GetID().String())
}

// Special node which will act as a root node for a tree
// Root node will always have Nil UUID i.e. 00000000-0000-0000-0000-000000000000
func CreateRootNode() *Node {
	return &Node{
		id:                NodeID(uuid.Nil.String()),
		nodeType:          ROOT,
		sibling:           nil,
		child:             nil,
		storageIdentifier: "",
		parent:            nil,
	}
}

// Various getter function for getting various properties of Node object
func (nodeObj *Node) GetID() NodeID {
	return nodeObj.id
}

func (nodeObj *Node) GetNodeType() NodeType {
	return nodeObj.nodeType
}

func (nodeObj *Node) GetIdentifier() rw.DataIdentifier {
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
