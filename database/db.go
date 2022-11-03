package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/dev-rodrigobaliza/go-blockchain/utils"
	"github.com/dgraph-io/badger"
)

const (
	dbFile   = "MANIFEST"
	dbLock   = "LOCK"
)

func DBexists(nodeId string) bool {
	path := checkBlockPath(nodeId)
	file := filepath.Join(path, dbFile)
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}

func GetDB(nodeId string) *badger.DB {
	path := checkBlockPath(nodeId)
	opts := badger.DefaultOptions(path)
	opts.EventLogging = false
	opts.Logger = nil

	db, err := open(path, opts)
	utils.Handle(err)

	return db
}

func open(dir string, opts badger.Options) (*badger.DB, error) {
	db, err := badger.Open(opts)
	if err != nil {
		if strings.Contains(err.Error(), dbLock) {
			db, err := retry(dir, opts)
			if err == nil {
				log.Println("database unlocked, value log truncated")
				return db, nil
			}

			log.Println("failed to unlock database:", err.Error())
		}

		return nil, err
	}

	return db, nil
}

func retry(dir string, originalOpts badger.Options) (*badger.DB, error) {
	lockPath := filepath.Join(dir, "LOCK")
	err := os.Remove(lockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to remove lock: %s", err.Error())
	}

	retryOpts := originalOpts
	retryOpts.Truncate = true

	return badger.Open(retryOpts)
}
