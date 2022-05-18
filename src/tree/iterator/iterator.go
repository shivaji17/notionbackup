package iterator

import (
	"errors"

	"github.com/sawantshivaji1997/notionbackup/src/tree/node"
)

// Done is returned by an iterator's Next method when the iteration is
// complete; when there are no more items to return.
// Every Iterator type extending Iterator interface must return this error
// when there are no more items to iterate
var Done = errors.New("no more items in iterator")

// Every Iterator type must extend this interface while implementing the
// iterator
type Iterator interface {
	Next() (*node.Node, error)
}
