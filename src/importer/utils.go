package importer

import (
	"encoding/json"
	"fmt"

	"github.com/jomei/notionapi"
	"github.com/shivaji17/notionbackup/src/utils"
)

func copyBlock(block notionapi.Block) notionapi.Block {
	dataBytes, _ := json.Marshal(block)

	var response map[string]interface{}
	json.Unmarshal(dataBytes, &response)

	newBlock, _ := utils.DecodeBlockObject(response)
	return newBlock
}

type objectUuidMapping struct {
	pageMap     map[notionapi.PageID]notionapi.PageID
	databaseMap map[notionapi.DatabaseID]notionapi.DatabaseID
	blockMap    map[notionapi.BlockID]notionapi.BlockID
}

func (o *objectUuidMapping) insertPageUuid(oldUuid,
	newUuid notionapi.ObjectID) {
	o.pageMap[notionapi.PageID(oldUuid)] = notionapi.PageID(newUuid)
}

func (o *objectUuidMapping) getPageUuid(
	oldUuid notionapi.PageID) (notionapi.PageID, error) {
	newUuid, found := o.pageMap[oldUuid]
	if !found {
		return "", fmt.Errorf("new uuid for page %s does not exist", oldUuid)
	}

	return newUuid, nil
}

func (o *objectUuidMapping) insertDatabaseUuid(oldUuid,
	newUuid notionapi.ObjectID) {
	o.databaseMap[notionapi.DatabaseID(oldUuid)] = notionapi.DatabaseID(newUuid)
}

func (o *objectUuidMapping) getDatabaseUuid(
	oldUuid notionapi.DatabaseID) (notionapi.DatabaseID, error) {
	newUuid, found := o.databaseMap[oldUuid]
	if !found {
		return "", fmt.Errorf("new uuid for database %s does not exist", oldUuid)
	}

	return newUuid, nil
}

func (o *objectUuidMapping) insertBlockUuid(oldUuid,
	newUuid notionapi.ObjectID) {
	o.blockMap[notionapi.BlockID(oldUuid)] = notionapi.BlockID(newUuid)
}

func (o *objectUuidMapping) getBlockUuid(
	oldUuid notionapi.BlockID) (notionapi.BlockID, error) {
	newUuid, found := o.blockMap[oldUuid]
	if !found {
		return "", fmt.Errorf("new uuid for block %s does not exist", oldUuid)
	}

	return newUuid, nil
}
