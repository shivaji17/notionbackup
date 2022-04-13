package utils

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/jomei/notionapi"
)

func ParsePageJsonString(jsonBytes []byte) (*notionapi.Page, error) {
	page := &notionapi.Page{}
	err := json.Unmarshal(jsonBytes, &page)
	if err != nil {
		return nil, err
	}
	return page, nil
}

func ParseSearchResponseJsonString(jsonBytes []byte) (*notionapi.SearchResponse, error) {
	searchResponse := &notionapi.SearchResponse{}
	err := json.Unmarshal(jsonBytes, &searchResponse)
	if err != nil {
		return nil, err
	}
	return searchResponse, nil
}

func ParseDatabaseJsonString(jsonBytes []byte) (*notionapi.Database, error) {
	database := &notionapi.Database{}
	err := json.Unmarshal(jsonBytes, &database)
	if err != nil {
		return nil, err
	}
	return database, nil
}

func CheckIfDirExists(dirPath string) error {
	_, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		return err
	}
	return nil
}

func CreateDirectory(dirPath string) error {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return err
	}

	return os.MkdirAll(absPath, 0700)
}

// Taken from https://github.com/jomei/notionapi/blob/main/block.go#L546
func DecodeBlockObject(raw map[string]interface{}) (notionapi.Block, error) {
	var b notionapi.Block
	switch notionapi.BlockType(raw["type"].(string)) {
	case notionapi.BlockTypeParagraph:
		b = &notionapi.ParagraphBlock{}
	case notionapi.BlockTypeHeading1:
		b = &notionapi.Heading1Block{}
	case notionapi.BlockTypeHeading2:
		b = &notionapi.Heading2Block{}
	case notionapi.BlockTypeHeading3:
		b = &notionapi.Heading3Block{}
	case notionapi.BlockCallout:
		b = &notionapi.CalloutBlock{}
	case notionapi.BlockQuote:
		b = &notionapi.QuoteBlock{}
	case notionapi.BlockTypeBulletedListItem:
		b = &notionapi.BulletedListItemBlock{}
	case notionapi.BlockTypeNumberedListItem:
		b = &notionapi.NumberedListItemBlock{}
	case notionapi.BlockTypeToDo:
		b = &notionapi.ToDoBlock{}
	case notionapi.BlockTypeCode:
		b = &notionapi.CodeBlock{}
	case notionapi.BlockTypeToggle:
		b = &notionapi.ToggleBlock{}
	case notionapi.BlockTypeChildPage:
		b = &notionapi.ChildPageBlock{}
	case notionapi.BlockTypeEmbed:
		b = &notionapi.EmbedBlock{}
	case notionapi.BlockTypeImage:
		b = &notionapi.ImageBlock{}
	case notionapi.BlockTypeVideo:
		b = &notionapi.VideoBlock{}
	case notionapi.BlockTypeFile:
		b = &notionapi.FileBlock{}
	case notionapi.BlockTypePdf:
		b = &notionapi.PdfBlock{}
	case notionapi.BlockTypeBookmark:
		b = &notionapi.BookmarkBlock{}
	case notionapi.BlockTypeChildDatabase:
		b = &notionapi.ChildDatabaseBlock{}
	case notionapi.BlockTypeTableOfContents:
		b = &notionapi.TableOfContentsBlock{}
	case notionapi.BlockTypeDivider:
		b = &notionapi.DividerBlock{}
	case notionapi.BlockTypeEquation:
		b = &notionapi.EquationBlock{}
	case notionapi.BlockTypeBreadcrumb:
		b = &notionapi.BreadcrumbBlock{}
	case notionapi.BlockTypeColumn:
		b = &notionapi.ColumnBlock{}
	case notionapi.BlockTypeColumnList:
		b = &notionapi.ColumnListBlock{}
	case notionapi.BlockTypeLinkPreview:
		b = &notionapi.LinkPreviewBlock{}
	case notionapi.BlockTypeLinkToPage:
		b = &notionapi.LinkToPageBlock{}
	case notionapi.BlockTypeTemplate:
		b = &notionapi.TemplateBlock{}
	case notionapi.BlockTypeSyncedBlock:
		b = &notionapi.SyncedBlock{}
	case notionapi.BlockTypeTableBlock:
		b = &notionapi.TableBlock{}
	case notionapi.BlockTypeTableRowBlock:
		b = &notionapi.TableRowBlock{}

	case notionapi.BlockTypeUnsupported:
		b = &notionapi.UnsupportedBlock{}
	default:
		return &notionapi.UnsupportedBlock{}, nil
	}
	j, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(j, b)
	return b, err
}
