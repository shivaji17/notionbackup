package iterator_test

import (
	"container/list"
	"context"
	"math/rand"
	"testing"

	"github.com/google/uuid"
	"github.com/jomei/notionapi"
	"github.com/sawantshivaji1997/notionbackup/src/mocks"
	"github.com/sawantshivaji1997/notionbackup/src/rw"
	"github.com/sawantshivaji1997/notionbackup/src/tree/iterator"
	"github.com/sawantshivaji1997/notionbackup/src/tree/node"
	"github.com/stretchr/testify/assert"
)

// Helper function to create a node object of any type (i.e. Database, Page or Block)
func getRandomNodeObject(t *testing.T) *node.Node {
	n := rand.Intn(3)
	mockedRW := mocks.NewReaderWriter(t)

	if n == 0 {
		mockedRW.On("WriteDatabase", context.Background(), &notionapi.Database{}).Return(rw.DataIdentifier(uuid.New().String()), nil)
		databaseNode, _ := node.CreateDatabaseNode(context.Background(), &notionapi.Database{}, mockedRW)
		assert.NotNil(t, databaseNode)
		return databaseNode
	} else if n == 1 {
		mockedRW.On("WritePage", context.Background(), &notionapi.Page{}).Return(rw.DataIdentifier(uuid.New().String()), nil)
		pageNode, _ := node.CreatePageNode(context.Background(), &notionapi.Page{}, mockedRW)
		assert.NotNil(t, pageNode)
		return pageNode
	}

	mockedRW.On("WriteBlock", context.Background(), &notionapi.ParagraphBlock{}).Return(rw.DataIdentifier(uuid.New().String()), nil)
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

func TestParentIterator(t *testing.T) {
	tests := []struct {
		name             string
		expectedChildren int
		passNilPointer   bool
	}{
		{
			name:             "Only one node present",
			expectedChildren: 0,
			passNilPointer:   false,
		},
		{
			name:             "Parent Present",
			expectedChildren: 5,
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
			expectedUUIDList = append(expectedUUIDList, nodeObj.GetID())
			for i := 1; i <= test.expectedChildren; i++ {
				childNode := getRandomNodeObject(t)
				nodeObj.AddChild(childNode)
				expectedUUIDList = append(expectedUUIDList, childNode.GetID())
				nodeObj = childNode
			}

			if test.passNilPointer {
				nodeObj = nil
				expectedUUIDList = []node.NodeID{}
			}

			iter := iterator.GetParentIterator(nodeObj)
			actualUUIDList := []node.NodeID{}

			for {
				obj, err := iter.Next()

				if err == iterator.Done {
					break
				}

				actualUUIDList = append([]node.NodeID{obj.GetID()}, actualUUIDList...)
			}

			assert.Equal(t, len(expectedUUIDList), len(actualUUIDList))
			assert.Equal(t, expectedUUIDList, actualUUIDList)
		})
	}

}
