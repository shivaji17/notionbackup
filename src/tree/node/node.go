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
	id         NodeID
	nodeType   NodeType
	sibling    *Node
	child      *Node
	identifier rw.DataIdentifier
}

func createNode(id NodeID, nodeType NodeType, identifier rw.DataIdentifier) (*Node, error) {
	return &Node{
		id:         id,
		nodeType:   nodeType,
		sibling:    nil,
		child:      nil,
		identifier: identifier,
	}, nil
}

func CreateDatabaseNode(ctx context.Context, database *notionapi.Database, rw rw.ReaderWriter) (*Node, error) {
	identifier, err := rw.WriteDatabase(ctx, database)
	if err != nil {
		return nil, err
	}

	return createNode(NodeID(uuid.New().String()), DATABASE, identifier)
}

func CreatePageNode(ctx context.Context, page *notionapi.Page, rw rw.ReaderWriter) (*Node, error) {
	identifier, err := rw.WritePage(ctx, page)
	if err != nil {
		return nil, err
	}

	return createNode(NodeID(uuid.New().String()), PAGE, identifier)
}

func CreateBlockNode(ctx context.Context, block notionapi.Block, rw rw.ReaderWriter) (*Node, error) {
	identifier, err := rw.WriteBlock(ctx, block)
	if err != nil {
		return nil, err
	}

	return createNode(NodeID(uuid.New().String()), BLOCK, identifier)
}

func CreateRootNode() *Node {
	return &Node{
		id:         NodeID(uuid.Nil.String()),
		nodeType:   ROOT,
		sibling:    nil,
		child:      nil,
		identifier: "",
	}
}

func (nodeObj *Node) GetID() NodeID {
	return nodeObj.id
}

func (nodeObj *Node) GetNodeType() NodeType {
	return nodeObj.nodeType
}

func (nodeObj *Node) GetIdentifier() rw.DataIdentifier {
	return nodeObj.identifier
}

func (nodeObj *Node) HasChildNode() bool {
	return nodeObj.child != nil
}

func (nodeObj *Node) GetChildNode() *Node {
	return nodeObj.child
}
