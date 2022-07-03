package iterator

import "github.com/sawantshivaji1997/notionbackup/src/tree/node"

// ChildIterator would iterate only it's children and not grandchildren
type ChildIterator struct {
	parentNode *node.Node
	currNode   *node.Node
}

func GetChildIterator(nodeObj *node.Node) Iterator {
	var currNode *node.Node

	if nodeObj != nil {
		currNode = nodeObj.GetChildNode()
	}

	return &ChildIterator{
		parentNode: nodeObj,
		currNode:   currNode,
	}
}

func (iter *ChildIterator) Next() (*node.Node, error) {
	if iter.parentNode != nil {
		temp := iter.currNode

		if temp == nil {
			return nil, ErrDone
		}

		iter.currNode = iter.currNode.GetSiblingNode()
		return temp, nil
	}

	return nil, ErrDone
}
