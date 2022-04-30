package iterator_test

import (
	"container/list"
	"context"
	"math/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/jomei/notionapi"
	"github.com/sawantshivaji1997/notionbackup/src/rw"
	"github.com/sawantshivaji1997/notionbackup/src/tree/iterator"
	"github.com/sawantshivaji1997/notionbackup/src/tree/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocking a ReaderWriter for Node Object
type MockedReaderWriter struct {
	mock.Mock
}

func (mobj *MockedReaderWriter) WriteDatabase(ctx context.Context, database *notionapi.Database) (rw.DataIdentifier, error) {
	args := mobj.Called(ctx, database)

	return rw.DataIdentifier(args.String(0)), args.Error(1)
}

func (mobj *MockedReaderWriter) ReadDatabase(ctx context.Context, identifier rw.DataIdentifier) (*notionapi.Database, error) {
	// Not needed
	return nil, nil
}

func (mobj *MockedReaderWriter) WritePage(ctx context.Context, page *notionapi.Page) (rw.DataIdentifier, error) {
	args := mobj.Called(ctx, page)

	return rw.DataIdentifier(args.String(0)), args.Error(1)
}

func (mobj *MockedReaderWriter) ReadPage(ctx context.Context, identifier rw.DataIdentifier) (*notionapi.Page, error) {
	// Not needed
	return nil, nil
}

func (mobj *MockedReaderWriter) WriteBlock(ctx context.Context, block notionapi.Block) (rw.DataIdentifier, error) {
	args := mobj.Called(ctx, block)

	return rw.DataIdentifier(args.String(0)), args.Error(1)
}

func (mobj *MockedReaderWriter) ReadBlock(ctx context.Context, identifier rw.DataIdentifier) (notionapi.Block, error) {
	// Not needed
	return nil, nil
}

// Helper function to create a node object of any type (i.e. Database, Page or Block)
func getRandomNodeObject(t *testing.T) *node.Node {
	n := rand.Intn(3)
	mockedRW := &MockedReaderWriter{}
	mockedRW.On("WriteDatabase", context.Background(), &notionapi.Database{}).Return(uuid.New().String(), nil)
	mockedRW.On("WritePage", context.Background(), &notionapi.Page{}).Return(uuid.New().String(), nil)
	mockedRW.On("WriteBlock", context.Background(), &notionapi.ParagraphBlock{}).Return(uuid.New().String(), nil)
	if n == 0 {
		databaseNode, _ := node.CreateDatabaseNode(context.Background(), &notionapi.Database{}, mockedRW)
		assert.NotNil(t, databaseNode)
		return databaseNode
	} else if n == 1 {
		pageNode, _ := node.CreatePageNode(context.Background(), &notionapi.Page{}, mockedRW)
		assert.NotNil(t, pageNode)
		return pageNode
	}
	blockNode, _ := node.CreateBlockNode(context.Background(), &notionapi.ParagraphBlock{}, mockedRW)
	assert.NotNil(t, blockNode)
	return blockNode
}

// helper function to 'childNodes' number of children to 'parentNode' node
func addchilds(t *testing.T, childNodes int, parentNode *node.Node, uuidList *[]node.NodeID) {
	if parentNode == nil {
		return
	}

	for i := 1; i <= childNodes; i++ {
		childNode := getRandomNodeObject(t)
		parentNode.AddChild(childNode)
		*uuidList = append(*uuidList, childNode.GetID())
	}
}

func TestChildIteration(t *testing.T) {
	tests := []struct {
		name             string
		expectedChildren int
		passNilPointer   bool
	}{
		{
			name:             "No child nodes present",
			expectedChildren: 0,
			passNilPointer:   false,
		},
		{
			name:             "More than one children exists",
			expectedChildren: 10,
			passNilPointer:   false,
		},
		{
			name:             "Only one child exists",
			expectedChildren: 1,
			passNilPointer:   false,
		},
		{
			name:             "Pass nil pointer",
			expectedChildren: 0,
			passNilPointer:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nodeObj := getRandomNodeObject(t)
			expectedUUIDList := []node.NodeID{}
			addchilds(t, test.expectedChildren, nodeObj, &expectedUUIDList)

			if test.passNilPointer {
				nodeObj = nil
			}

			iter := iterator.GetChildIterator(nodeObj)
			actualUUIDList := []node.NodeID{}

			for {
				obj, err := iter.Next()

				if err == iterator.Done {
					break
				}

				actualUUIDList = append(actualUUIDList, obj.GetID())
			}

			assert.Equal(t, len(expectedUUIDList), len(actualUUIDList))
			assert.Equal(t, expectedUUIDList, actualUUIDList)
		})
	}
}

// Helper function to buid tree in breadth first manner
// This function would build the tree having atleast 'totalNodes' total nodes
// atmost 'expectedChildren' childs
func createTree(t *testing.T, totalNodes int, expectedChildren int, parentNode *node.Node, uuidList *[]node.NodeID) {

	if parentNode == nil {
		return
	}

	queue := list.New()
	queue.PushBack(parentNode)
	for {
		if queue.Len() == 0 {
			break
		}

		if len(*uuidList) >= totalNodes {
			break
		}

		front := queue.Front()
		currNode, ok := front.Value.(*node.Node)
		assert.True(t, ok)
		for i := 1; i <= expectedChildren; i++ {
			childNode := getRandomNodeObject(t)
			currNode.AddChild(childNode)
			*uuidList = append(*uuidList, childNode.GetID())
			queue.PushBack(childNode)
		}
	}

}

func TestTreeIterator(t *testing.T) {
	tests := []struct {
		name             string
		expectedChildren int
		totalNodes       int
		passNilPointer   bool
		passAsRootNode   bool
	}{
		{
			name:             "Pass root node as nil pointer",
			expectedChildren: 0,
			totalNodes:       0,
			passNilPointer:   true,
			passAsRootNode:   false,
		},
		{
			name:             "Pass valid root node",
			expectedChildren: 8,
			totalNodes:       108,
			passNilPointer:   false,
			passAsRootNode:   true,
		},
		{
			name:             "Pass non root node",
			expectedChildren: 5,
			totalNodes:       75,
			passNilPointer:   false,
			passAsRootNode:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var rootNode *node.Node
			expectedUUIDList := []node.NodeID{}
			if !test.passNilPointer {
				if test.passAsRootNode {
					rootNode = node.CreateRootNode()
					assert.NotNil(t, rootNode)
				} else {
					rootNode = getRandomNodeObject(t)
					expectedUUIDList = append(expectedUUIDList, rootNode.GetID())
				}
			}

			createTree(t, test.totalNodes, test.expectedChildren, rootNode, &expectedUUIDList)

			treeIter := iterator.GetTreeIterator(rootNode)
			actualUUIDList := []node.NodeID{}
			for {
				obj, err := treeIter.Next()

				if err == iterator.Done {
					break
				}
				actualUUIDList = append(actualUUIDList, obj.GetID())
			}

			assert.Equal(t, len(expectedUUIDList), len(actualUUIDList))
			assert.Equal(t, expectedUUIDList, actualUUIDList)
		})
	}
}
