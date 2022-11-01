package blockchain

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dgraph-io/badger"
)

const (
	dbPath = "db/blocks"
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

func InitBlockChain() *BlockChain {
	curPath, err := os.Getwd()
	Handle(err)

	path := filepath.Join(curPath, dbPath)
	opts := badger.DefaultOptions(path)
	db, err := badger.Open(opts)
	Handle(err)

	var lastHash []byte
	err = db.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get([]byte("lh")); err == badger.ErrKeyNotFound {
			fmt.Println("no existing blockchain found on the database")
			genesis := Genesis()
			fmt.Println("genesis proved")

			err = txn.Set(genesis.Hash, genesis.Serialize())
			Handle(err)

			err = txn.Set([]byte("lh"), genesis.Hash)
			Handle(err)

			lastHash = genesis.Hash
		} else {
			item, err := txn.Get([]byte("lh"))
			Handle(err)

			err = item.Value(func(val []byte) error {
				lastHash = append([]byte{}, val...)

				return nil
			})
			Handle(err)
		}

		return err
	})
	Handle(err)

	blockChain := BlockChain{lastHash, db}

	return &blockChain
}

func (chain *BlockChain) AddBlock(data string) {
	var lastHash []byte

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handle(err)

		err = item.Value(func(val []byte) error {
			lastHash = append([]byte{}, val...)

			return nil
		})

		return err
	})
	Handle(err)

	newBlock := CreateBlock(data, lastHash)

	err = chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		Handle(err)

		err = txn.Set([]byte("lh"), newBlock.Hash)

		chain.LastHash = newBlock.Hash

		return err
	})
	Handle(err)
}

func (chain *BlockChain) Iterator() *BlockChainIterator {
	iter := &BlockChainIterator{chain.LastHash, chain.Database}

	return iter
}
