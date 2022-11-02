package blockchain

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/dev-rodrigobaliza/go-blockchain/utils"
	"github.com/dgraph-io/badger"
)

const (
	dbPath      = "db"
	dbFile      = "db/MANIFEST"
	genesisData = "First Transaction from Genesis"
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

func CheckDBPath() string {
	curPath, err := os.Getwd()
	utils.Handle(err)

	path := filepath.Join(curPath, dbPath)
	_ = os.Mkdir(path, os.ModePerm)

	return path
}

func DBexists() bool {
	_, err := os.Stat(dbFile)
	return !os.IsNotExist(err)
}

func GetDB() *badger.DB {
	path := CheckDBPath()
	opts := badger.DefaultOptions(path)
	db, err := badger.Open(opts)
	utils.Handle(err)

	return db
}

func InitBlockChain(address string) *BlockChain {
	if DBexists() {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}

	db := GetDB()

	var lastHash []byte
	err := db.Update(func(txn *badger.Txn) error {
		cbtx := CoinbaseTx(address, genesisData)
		genesis := Genesis(cbtx)
		fmt.Println("Genesis created")
		err := txn.Set(genesis.Hash, genesis.Serialize())
		utils.Handle(err)
		err = txn.Set([]byte("lh"), genesis.Hash)

		lastHash = genesis.Hash

		return err
	})
	utils.Handle(err)

	blockChain := BlockChain{lastHash, db}

	return &blockChain
}

func ContinueBlockChain(address string) *BlockChain {
	if !DBexists() {
		fmt.Println("No existing blockchain, create one!")
		runtime.Goexit()
	}

	db := GetDB()

	var lastHash []byte
	err := db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		utils.Handle(err)

		err = item.Value(func(val []byte) error {
			lastHash = append([]byte{}, val...)

			return nil
		})
		return err
	})
	utils.Handle(err)

	blockChain := BlockChain{lastHash, db}

	return &blockChain
}

func (chain *BlockChain) AddBlock(transactions []*Transaction) {
	var lastHash []byte

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		utils.Handle(err)

		err = item.Value(func(val []byte) error {
			lastHash = append([]byte{}, val...)

			return nil
		})

		return err
	})
	utils.Handle(err)

	newBlock := CreateBlock(transactions, lastHash)

	err = chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		utils.Handle(err)

		err = txn.Set([]byte("lh"), newBlock.Hash)

		chain.LastHash = newBlock.Hash

		return err
	})
	utils.Handle(err)
}

func (chain *BlockChain) Iterator() *BlockChainIterator {
	iter := &BlockChainIterator{chain.LastHash, chain.Database}

	return iter
}

func (chain *BlockChain) FindUnspentTransactions(address string) []Transaction {
	var unspentTxs []Transaction

	spentTXOs := make(map[string][]int)

	iter := chain.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

			Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}

				if out.CanBeUnlocked(address) {
					unspentTxs = append(unspentTxs, *tx)
				}
			}

			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					if in.CanUnlock(address) {
						inTXID := hex.EncodeToString(in.ID)
						spentTXOs[inTXID] = append(spentTXOs[inTXID], in.Out)
					}
				}
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return unspentTxs
}

func (chain *BlockChain) FindUTXO(address string) []TxOutput {
	var UTXOs []TxOutput

	unspentTransactions := chain.FindUnspentTransactions(address)

	for _, tx := range unspentTransactions {
		for _, out := range tx.Outputs {
			if out.CanBeUnlocked(address) {
				UTXOs = append(UTXOs, out)
			}
		}
	}

	return UTXOs

}

func (chain *BlockChain) FindSpendableOutputs(address string, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	unspentTxs := chain.FindUnspentTransactions(address)
	accumulated := 0

	Work:
	for _, tx := range unspentTxs {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Outputs {
			if out.CanBeUnlocked(address) && accumulated < amount {
				accumulated += out.Value
				unspentOuts[txID] = append(unspentOuts[txID], outIdx)

				if accumulated >= amount {
					break Work
				}
			}
		}

	}

	return accumulated, unspentOuts
}