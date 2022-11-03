package database

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dev-rodrigobaliza/go-blockchain/utils"
)

const (
	databasePath = "db"
	blocksPath   = "blocks_%s"
)

func checkBlockPath(nodeId string) string {
	systemPath := utils.CheckSystemPath()
	dbPath := filepath.Join(systemPath, databasePath)
	_ = os.Mkdir(dbPath, os.ModePerm)

	dbName := fmt.Sprintf(blocksPath, nodeId)
	blockPath := filepath.Join(dbPath, dbName)

	return blockPath
}
