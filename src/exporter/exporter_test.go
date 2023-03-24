package exporter_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jomei/notionapi"
	"github.com/shivaji17/notionbackup/src/exporter"
	"github.com/shivaji17/notionbackup/src/metadata"
	"github.com/shivaji17/notionbackup/src/mocks"
	"github.com/shivaji17/notionbackup/src/rw"
	"github.com/shivaji17/notionbackup/src/tree"
	"github.com/shivaji17/notionbackup/src/tree/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func createNode(t *testing.T, nodeType node.NodeType) *node.Node {
	mockedRW := mocks.NewReaderWriter(t)

	if nodeType == node.DATABASE {
		mockedRW.On("WriteDatabase", context.Background(), &notionapi.Database{}).
			Return(rw.DataIdentifier(uuid.New().String()), nil)
		databaseNode, _ := node.CreateDatabaseNode(context.Background(),
			&notionapi.Database{}, mockedRW)
		assert.NotNil(t, databaseNode)
		return databaseNode
	} else if nodeType == node.PAGE {
		mockedRW.On("WritePage", context.Background(), &notionapi.Page{}).
			Return(rw.DataIdentifier(uuid.New().String()), nil)
		pageNode, _ := node.CreatePageNode(context.Background(),
			&notionapi.Page{}, mockedRW)
		assert.NotNil(t, pageNode)
		return pageNode
	} else if nodeType == node.BLOCK {
		mockedRW.On("WriteBlock", context.Background(),
			&notionapi.ParagraphBlock{}).Return(rw.DataIdentifier(
			uuid.New().String()), nil)
		blockNode, _ := node.CreateBlockNode(context.Background(),
			&notionapi.ParagraphBlock{}, mockedRW)
		assert.NotNil(t, blockNode)
		return blockNode
	}

	return nil
}

func TestExportTree(t *testing.T) {

	assert := assert.New(t)
	//////////////////////////////////////////////////////////////////////////////
	t.Run("Valid root node with children", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		expectedStorageConfig := &metadata.StorageConfig{
			Config: &metadata.StorageConfig_Local_{
				Local: &metadata.StorageConfig_Local{
					BlocksDir:   rw.BLOCK_DIR_NAME,
					PageDir:     rw.PAGE_DIR_NAME,
					DatabaseDir: rw.DATABASE_DIR_NAME,
				},
			},
		}

		mockedRW.On("GetStorageConfig", context.Background()).
			Return(expectedStorageConfig, nil)
		mockedRW.On("WriteMetaData", context.Background(), mock.Anything).
			Return(nil)

		rootNode := node.CreateRootNode()
		pageNode1 := createNode(t, node.PAGE)
		pageNode1.AddChild(createNode(t, node.BLOCK))
		pageNode1.AddChild(createNode(t, node.BLOCK))

		rootNode.AddChild(pageNode1)

		pageNode2 := createNode(t, node.PAGE)
		pageNode2.AddChild(createNode(t, node.BLOCK))
		databaseNode := createNode(t, node.DATABASE)
		databaseNode.AddChild(createNode(t, node.PAGE))
		databaseNode.AddChild(createNode(t, node.PAGE))
		databaseNode.AddChild(pageNode2)

		rootNode.AddChild(databaseNode)

		rootNode.AddChild(createNode(t, node.DATABASE))
		rootNode.AddChild(createNode(t, node.PAGE))

		tree := &tree.Tree{RootNode: rootNode}
		err := exporter.ExportTree(context.Background(), mockedRW, tree)
		assert.Nil(err)
	})

	//////////////////////////////////////////////////////////////////////////////
	t.Run("Error in getting storage config", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		mockedRW.On("GetStorageConfig", context.Background()).
			Return(nil, fmt.Errorf("failure"))

		rootNode := node.CreateRootNode()
		pageNode1 := createNode(t, node.PAGE)
		pageNode1.AddChild(createNode(t, node.BLOCK))
		pageNode1.AddChild(createNode(t, node.BLOCK))

		rootNode.AddChild(pageNode1)

		pageNode2 := createNode(t, node.PAGE)
		pageNode2.AddChild(createNode(t, node.BLOCK))
		databaseNode := createNode(t, node.DATABASE)
		databaseNode.AddChild(createNode(t, node.PAGE))
		databaseNode.AddChild(createNode(t, node.PAGE))
		databaseNode.AddChild(pageNode2)

		rootNode.AddChild(databaseNode)

		rootNode.AddChild(createNode(t, node.DATABASE))
		rootNode.AddChild(createNode(t, node.PAGE))

		tree := &tree.Tree{RootNode: rootNode}
		err := exporter.ExportTree(context.Background(), mockedRW, tree)
		assert.NotNil(err)
	})

	//////////////////////////////////////////////////////////////////////////////
	t.Run("Invalid root node", func(t *testing.T) {
		mockedRW := mocks.NewReaderWriter(t)
		rootNode := createNode(t, node.DATABASE)
		tree := &tree.Tree{RootNode: rootNode}
		err := exporter.ExportTree(context.Background(), mockedRW, tree)
		assert.NotNil(err)
	})
}
