package builder

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sawantshivaji1997/notionbackup/src/metadata"
	"github.com/sawantshivaji1997/notionbackup/src/tree"
	"github.com/sawantshivaji1997/notionbackup/src/tree/node"
)

type MetadataTreeBuilder struct {
	metadataCfg *metadata.MetaData
	nodeMap     map[node.NodeID]*node.Node
}

func GetMetaDataTreeBuilder(ctx context.Context,
	metadataCfg *metadata.MetaData) TreeBuilder {
	return &MetadataTreeBuilder{
		metadataCfg: metadataCfg,
		nodeMap:     make(map[node.NodeID]*node.Node),
	}
}

func (builder *MetadataTreeBuilder) addChildNodes(ctx context.Context,
	parentNode *node.Node, childList *metadata.ChildrenNotionObjectUuids) error {
	for _, child := range childList.ChildrenUuidList {
		childNode, found := builder.nodeMap[node.NodeID(child)]
		if !found {
			return fmt.Errorf("node with id '%s' does not exist", child)
		}

		parentNode.AddChild(childNode)
	}

	return nil
}

func (builder *MetadataTreeBuilder) buildTree(ctx context.Context) error {
	parent2ChildListMap := builder.metadataCfg.GetParentUuid_2ChildrenUuidMap()
	for parent, childList := range parent2ChildListMap {
		parentNode, found := builder.nodeMap[node.NodeID(parent)]
		if !found {
			return fmt.Errorf("node with id '%s' does not exist", parent)
		}

		err := builder.addChildNodes(ctx, parentNode, childList)
		if err != nil {
			return err
		}
	}
	return nil
}

func (builder *MetadataTreeBuilder) CreateNodes(ctx context.Context) error {
	for _, notionObj := range builder.metadataCfg.NotionObjectMap {
		nodeObj, err := node.CreateNode(notionObj)
		if err != nil {
			return err
		}

		builder.nodeMap[nodeObj.GetID()] = nodeObj
	}
	return nil
}

func (builder *MetadataTreeBuilder) BuildTree(ctx context.Context) (*tree.Tree,
	error) {
	err := builder.CreateNodes(ctx)
	if err != nil {
		return nil, err
	}

	err = builder.buildTree(ctx)
	if err != nil {
		return nil, err
	}

	rootNode, found := builder.nodeMap[node.NodeID(uuid.Nil.String())]
	if !found {
		return nil, fmt.Errorf("root node does not exist")
	}

	return &tree.Tree{
		RootNode: rootNode,
	}, nil
}
