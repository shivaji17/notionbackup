package iterator

import (
	"fmt"

	"github.com/shivaji17/notionbackup/src/tree/node"
)

// Done is returned by an iterator's Next method when the iteration is
// complete; when there are no more items to return.
// Every Iterator type extending Iterator interface must return this error
// when there are no more items to iterate
var ErrDone = fmt.Errorf("no more items in iterator")

// Every Iterator type must extend this interface while implementing the
// iterator
type Iterator interface {
	Next() (*node.Node, error)
}
