package exporter

import (
	"context"
	"errors"

	"github.com/rs/zerolog"
	"github.com/sawantshivaji1997/notionbackup/src/metadata"
	"github.com/sawantshivaji1997/notionbackup/src/rw"
	"github.com/sawantshivaji1997/notionbackup/src/tree"
	"github.com/sawantshivaji1997/notionbackup/src/tree/iterator"
	"github.com/sawantshivaji1997/notionbackup/src/tree/node"
)

func Convert2ProtoNotionObject(nodeObj *node.Node) (*metadata.NotionObject,
	error) {
	notionObj := &metadata.NotionObject{}
	notionObj.Uuid = nodeObj.GetID().String()
	notionObj.StorageIdentifier = nodeObj.GetStorageIdentifier().String()
	notionObj.NotionObjectId = nodeObj.GetNotionObjectId()

	switch nodeObj.GetNodeType() {
	case node.ROOT:
		notionObj.Type = metadata.NotionObjectType_ROOT
	case node.PAGE:
		notionObj.Type = metadata.NotionObjectType_PAGE
	case node.DATABASE:
		notionObj.Type = metadata.NotionObjectType_DATABASE
	case node.BLOCK:
		notionObj.Type = metadata.NotionObjectType_BLOCK
	default:
		return nil, errors.New("unknown notion object type of node")
	}

	return notionObj, nil
}

func GetChildrenUuidList(nodeObj *node.Node) []string {
	iter := iterator.GetChildIterator(nodeObj)
	childrenUuidList := []string{}
	for {
		childObj, err := iter.Next()
		if err == iterator.Done {
			break
		}
		childrenUuidList = append(childrenUuidList, childObj.GetID().String())
	}

	return childrenUuidList
}

func ExportTree(ctx context.Context, rw rw.ReaderWriter,
	tree *tree.Tree) error {
	log := zerolog.Ctx(ctx)
	if tree.RootNode.GetNodeType() != node.ROOT {
		errMsg := "root node does not have a type 'ROOT'"
		log.Error().Msg(errMsg)
		return errors.New(errMsg)
	}

	metadataObj := &metadata.MetaData{
		NotionObjectMap: make(map[string]*metadata.NotionObject),
		ObjectMapping:   make(map[string]*metadata.ChildrenNotionObjectUuids),
	}
	rootNodeNotionObject, err := Convert2ProtoNotionObject(tree.RootNode)
	if err != nil {
		return err
	}

	metadataObj.NotionObjectMap[tree.RootNode.GetID().String()] =
		rootNodeNotionObject

	iter := iterator.GetTreeIterator(tree.RootNode)
	for {
		nodeObj, err := iter.Next()
		if err == iterator.Done {
			break
		}

		notionObj, err := Convert2ProtoNotionObject(nodeObj)
		if err != nil {
			return err
		}

		metadataObj.NotionObjectMap[nodeObj.GetID().String()] = notionObj

		childrenUuidList := GetChildrenUuidList(nodeObj)
		if len(childrenUuidList) > 0 {
			metadataObj.ObjectMapping[nodeObj.GetID().String()] =
				&metadata.ChildrenNotionObjectUuids{
					ChildrenUuidList: childrenUuidList,
				}
		}
	}

	return rw.WriteMetaData(ctx, metadataObj)
}
