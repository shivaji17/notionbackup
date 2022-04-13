package rw

import (
	"context"

	"github.com/jomei/notionapi"
)

type DataIdentifier string

type ReaderWriter interface {
	WriteDatabase(context.Context, *notionapi.Database) (DataIdentifier, error)
	ReadDatabase(context.Context, DataIdentifier) (*notionapi.Database, error)
	WritePage(context.Context, *notionapi.Page) (DataIdentifier, error)
	ReadPage(context.Context, DataIdentifier) (*notionapi.Page, error)
	WriteBlock(context.Context, notionapi.Block) (DataIdentifier, error)
	ReadBlock(context.Context, DataIdentifier) (notionapi.Block, error)
}
