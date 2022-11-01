package blockchain

import "github.com/dgraph-io/badger"

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func (iter *BlockChainIterator) Next() *Block {
	var block *Block

	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		Handle(err)

		err = item.Value(func(val []byte) error {
			block = block.Deserialize(val)

			return nil
		})

		return err
	})
	Handle(err)

	iter.CurrentHash = block.PrevHash

	return block
}