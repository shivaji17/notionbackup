package builder

import (
	"context"
	"fmt"

	"github.com/sawantshivaji1997/notionbackup/src/tree"
	"github.com/sawantshivaji1997/notionbackup/src/tree/node"
)

type TreeBuilderRequest struct {
	PageIdList     []string
	DatabaseIdList []string
}
type TreeBuilder interface {
	BuildTree(context.Context) (*tree.Tree, error)
}

type stackContent struct {
	nodeObject *node.Node
	objectId   string
}

var stackEmpty = fmt.Errorf("no more items in stack")

type stack []*node.Node

// IsEmpty: check if stack is empty
func (s *stack) IsEmpty() bool {
	return len(*s) == 0
}

// Push a new value onto the stack
func (s *stack) Push(object *node.Node) {
	*s = append(*s, object)
}

// Remove and return top element of stack. Return false if stack is empty.
func (s *stack) Pop() (*node.Node, error) {
	if s.IsEmpty() {
		return nil, stackEmpty
	} else {
		index := len(*s) - 1
		object := (*s)[index]
		*s = (*s)[:index]
		return object, nil
	}
}
