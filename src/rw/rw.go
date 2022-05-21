package rw

import (
	"context"

	"github.com/jomei/notionapi"
)

type DataIdentifier string

func (d DataIdentifier) String() string {
	return string(d)
}

type ReaderWriter interface {
	WriteDatabase(context.Context, *notionapi.Database) (DataIdentifier, error)
	ReadDatabase(context.Context, DataIdentifier) (*notionapi.Database, error)
	WritePage(context.Context, *notionapi.Page) (DataIdentifier, error)
	ReadPage(context.Context, DataIdentifier) (*notionapi.Page, error)
	WriteBlock(context.Context, notionapi.Block) (DataIdentifier, error)
	ReadBlock(context.Context, DataIdentifier) (notionapi.Block, error)
	CleanUp(context.Context) error
}
