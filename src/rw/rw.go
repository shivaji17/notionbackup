package rw

import "github.com/jomei/notionapi"

type DataIdentifier string

type ReaderWriter interface {
	WriteDatabase(*notionapi.Database) (DataIdentifier, error)
	ReadDatabase(DataIdentifier) (*notionapi.Database, error)
	WritePage(*notionapi.Page) (DataIdentifier, error)
	ReadPage(DataIdentifier) (*notionapi.Page, error)
	WriteBlock(notionapi.Block) (DataIdentifier, error)
	ReadBlock(DataIdentifier) (notionapi.Block, error)
}
