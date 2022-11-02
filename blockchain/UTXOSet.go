package blockchain

import (
	"bytes"
	"encoding/hex"

	"github.com/dev-rodrigobaliza/go-blockchain/utils"
	"github.com/dgraph-io/badger"
)

var (
	utxoPrefix   = []byte("utxo-")
	prefixLength = len(utxoPrefix)
)

type UTXOSet struct {
	Blockchain *BlockChain
}

func (u *UTXOSet) Reindex() {
	db := u.Blockchain.Database

	u.DeleteByPrefix(utxoPrefix)

	UTXO := u.Blockchain.FindUTXO()

	err := db.Update(func(txn *badger.Txn) error {
		for txId, outs := range UTXO {
			key, err := hex.DecodeString(txId)
			if err != nil {
				return err
			}
			key = append(utxoPrefix, key...)

			err = txn.Set(key, outs.toBytes())
			utils.Handle(err)
		}

		return nil
	})
	utils.Handle(err)
}

func (u *UTXOSet) Update(block *Block) {
	db := u.Blockchain.Database

	err := db.Update(func(txn *badger.Txn) error {
		for _, tx := range block.Transactions {
			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					updatedOuts := TxOutputs{}
					inID := append(utxoPrefix, in.ID...)
					item, err := txn.Get(inID)
					utils.Handle(err)
					var outs TxOutputs
					err = item.Value(func(val []byte) error {
						return outs.fromBytes(val)
					})
					utils.Handle(err)

					for outIdx, out := range outs.Outputs {
						if outIdx != in.Out {
							updatedOuts.Outputs = append(updatedOuts.Outputs, out)
						}
					}

					if len(updatedOuts.Outputs) == 0 {
						err := txn.Delete(inID)
						if err != nil {
							utils.Handle(err)
						}
					}

					err = txn.Set(inID, updatedOuts.toBytes())
					utils.Handle(err)
				}
			}

			newOutputs := TxOutputs{}
			newOutputs.Outputs = append(newOutputs.Outputs, tx.Outputs...)

			txID := append(utxoPrefix, tx.ID...)
			err := txn.Set(txID, newOutputs.toBytes())
			utils.Handle(err)
		}

		return nil
	})
	utils.Handle(err)
}

func (u *UTXOSet) FindUnspentTransactions(pubKeyHash []byte) []*TxOutput {
	var UTXOs []*TxOutput

	db := u.Blockchain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			var outs TxOutputs
			item := it.Item()
			err := item.Value(func(val []byte) error {
				return outs.fromBytes(val)
			})
			utils.Handle(err)

			for _, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) {
					UTXOs = append(UTXOs, out)
				}
			}
		}

		return nil
	})
	utils.Handle(err)

	return UTXOs
}

func (u *UTXOSet) FindSpendableOutputs(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	accumulated := 0

	db := u.Blockchain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			var outs TxOutputs
			item := it.Item()
			err := item.Value(func(val []byte) error {
				return outs.fromBytes(val)
			})
			utils.Handle(err)
			id := item.Key()
			id = bytes.TrimPrefix(id, utxoPrefix)
			txID := hex.EncodeToString(id)

			for outIdx, out := range outs.Outputs {
				if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
					accumulated += out.Value
					unspentOuts[txID] = append(unspentOuts[txID], outIdx)
				}
			}

		}

		return nil
	})
	utils.Handle(err)

	return accumulated, unspentOuts
}

func (u *UTXOSet) CountTransactions() int {
	db := u.Blockchain.Database
	counter := 0

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(utxoPrefix); it.ValidForPrefix(utxoPrefix); it.Next() {
			counter++
		}

		return nil
	})
	utils.Handle(err)

	return counter
}

func (u *UTXOSet) DeleteByPrefix(prefix []byte) {
	deleteKeys := func(keysForDelete [][]byte) error {
		err := u.Blockchain.Database.Update(func(txn *badger.Txn) error {
			for _, key := range keysForDelete {
				err := txn.Delete(key)
				if err != nil {
					return err
				}
			}

			return nil
		})

		return err
	}

	collectSize := 100000
	u.Blockchain.Database.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		keysForDelete := make([][]byte, 0, collectSize)
		keysCollected := 0
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := it.Item().KeyCopy(nil)
			keysForDelete = append(keysForDelete, key)
			keysCollected++
			if keysCollected == collectSize {
				err := deleteKeys(keysForDelete)
				if err != nil {
					utils.Handle(err)
				}
				keysForDelete = make([][]byte, 0, collectSize)
				keysCollected = 0
			}
		}

		if keysCollected > 0 {
			err := deleteKeys(keysForDelete)
			if err != nil {
				utils.Handle(err)
			}
		}

		return nil
	})
}
