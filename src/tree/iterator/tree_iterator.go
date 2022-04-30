package iterator

import (
	"container/list"
	"errors"

	"github.com/google/uuid"
	"github.com/sawantshivaji1997/notionbackup/src/tree/node"
)

// TreeIterator would iterate the whole tree or subtree depending on the node
// passed to the iterator.
// If root node is passed, whole tree would be iterated else the subtree of
// given node would be iterated
type TreeIterator struct {
	queue *list.List
}

func GetTreeIterator(nodeObj *node.Node) Iterator {
	queue := list.New()

	if nodeObj != nil {
		if nodeObj.GetID() == node.NodeID(uuid.Nil.String()) {
			childIter := GetChildIterator(nodeObj)
			for {
				childNode, err := childIter.Next()
				if err == Done {
					break
				}
				queue.PushBack(childNode)
			}

		} else {
			queue.PushBack(nodeObj)
		}
	}

	return &TreeIterator{
		queue: queue,
	}
}

func (iter *TreeIterator) Next() (*node.Node, error) {
	if iter.queue.Len() == 0 {
		return nil, Done
	}

	front := iter.queue.Front()
	currNode, ok := front.Value.(*node.Node)
	if !ok {
		return nil, errors.New("failed to parse node object type")
	}

	childIter := GetChildIterator(currNode)
	iter.queue.Remove(front)

	for {
		childNode, err := childIter.Next()
		if err == Done {
			break
		}
		iter.queue.PushBack(childNode)
	}

	return currNode, nil
}
