package iterator

import "github.com/sawantshivaji1997/notionbackup/src/tree/node"

type ParentIterator struct {
	currentNode *node.Node
}

func GetParentIterator(nodeObj *node.Node) *ParentIterator {
	return &ParentIterator{
		currentNode: nodeObj,
	}
}

func (iter *ParentIterator) Next() (*node.Node, error) {
	if iter.currentNode != nil {
		temp := iter.currentNode
		iter.currentNode = iter.currentNode.GetParentNode()
		return temp, nil
	}

	return nil, Done
}
