package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"runtime"

	"github.com/dev-rodrigobaliza/go-blockchain/database"
	"github.com/dev-rodrigobaliza/go-blockchain/utils"
	"github.com/dgraph-io/badger"
)

const (
	genesisData = "First Transaction from Genesis"
	lastHashPrefix = "lh"
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

func InitBlockChain(address, nodeId string) *BlockChain {
	if database.DBexists(nodeId) {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}

	db := database.GetDB(nodeId)

	var lastHash []byte
	err := db.Update(func(txn *badger.Txn) error {
		cbtx := CoinbaseTx(address, genesisData)
		genesis := Genesis(cbtx)
		fmt.Println("Genesis created")
		err := txn.Set(genesis.Hash, genesis.Serialize())
		utils.Handle(err)
		err = txn.Set([]byte(lastHashPrefix), genesis.Hash)

		lastHash = genesis.Hash

		return err
	})
	utils.Handle(err)

	blockChain := BlockChain{lastHash, db}

	return &blockChain
}

func ContinueBlockChain(nodeId string) *BlockChain {
	if !database.DBexists(nodeId) {
		fmt.Println("No existing blockchain, create one!")
		runtime.Goexit()
	}

	db := database.GetDB(nodeId)

	var lastHash []byte
	err := db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(lastHashPrefix))
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

func (chain *BlockChain) AddBlock(block *Block) {
	var lastHash []byte
	var lastBlock *Block

	err := chain.Database.Update(func(txn *badger.Txn) error {
		_, err := txn.Get(block.Hash)
		if err == nil {
			return nil
		}

		blockData := block.Serialize()
		err = txn.Set(block.Hash, blockData)
		utils.Handle(err)

		item, err := txn.Get([]byte(lastHashPrefix))
		utils.Handle(err)

		err = item.Value(func(val []byte) error {
			lastHash = append([]byte{}, val...)

			return nil
		})
		utils.Handle(err)

		item, err = txn.Get(lastHash)
		utils.Handle(err)

		err = item.Value(func(val []byte) error {
			lastBlock.Deserialize(val)

			return nil
		})
		utils.Handle(err)

		if block.Height > lastBlock.Height {
			err = txn.Set([]byte(lastHashPrefix), block.Hash)
			utils.Handle(err)

			chain.LastHash = block.Hash
		}

		return nil
	})
	utils.Handle(err)
}

func (chain *BlockChain) GetBlock(blockHash []byte) (*Block, error) {
	var block *Block

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(blockHash)
		if err != nil {
			return errors.New("block not found")
		}

		err = item.Value(func(val []byte) error {
			block.Deserialize(val)

			return nil
		})

		return err
	})
	if err != nil {
		return nil, err
	}

	return block, nil
}

func (chain *BlockChain) GetBestHeight() int {
	var lastHash []byte
	var lastBlock *Block
	var lastHeight int

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(lastHashPrefix))
		utils.Handle(err)

		err = item.Value(func(val []byte) error {
			lastHash = append([]byte{}, val...)

			return nil
		})
		utils.Handle(err)

		item, err = txn.Get(lastHash)
		utils.Handle(err)

		err = item.Value(func(val []byte) error {
			lastBlock.Deserialize(val)

			return nil
		})
		utils.Handle(err)

		lastHeight = lastBlock.Height

		return err
	})
	utils.Handle(err)

	return lastHeight

}

func (chain *BlockChain) GetBlockHashes() [][]byte {
	var blocks [][]byte

	iter := chain.Iterator()

	for {
		block := iter.Next()

		blocks = append(blocks, block.Hash)

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return blocks
}

func (chain *BlockChain) MineBlock(transactions []Transaction) Block {
	var lastHash []byte

	lastHeight := chain.GetBestHeight()
	lastHeight++
	newBlock := NewBlock(transactions, lastHash, lastHeight)

	err := chain.Database.Update(func(txn *badger.Txn) error {
		err := txn.Set(newBlock.Hash, newBlock.Serialize())
		utils.Handle(err)

		err = txn.Set([]byte(lastHashPrefix), newBlock.Hash)

		chain.LastHash = newBlock.Hash

		return err
	})
	utils.Handle(err)

	return newBlock
}

func (chain *BlockChain) Iterator() *BlockChainIterator {
	iter := &BlockChainIterator{chain.LastHash, chain.Database}

	return iter
}

func (chain *BlockChain) FindUnspentTransactions(pubKeyHash []byte) []Transaction {
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

				if out.IsLockedWithKey(pubKeyHash) {
					unspentTxs = append(unspentTxs, tx)
				}
			}

			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					if in.UsesKey(pubKeyHash) {
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

func (chain *BlockChain) FindUTXO() map[string]TxOutputs {
	UTXO := make(map[string]TxOutputs)
	spentTXOs := make(map[string][]int)

	iter := chain.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTXOs[txID] != nil {
					for _, spentOutIdx := range spentTXOs[txID] {
						if spentOutIdx == outIdx {
							continue Outputs
						}
					}
				}

				outs := UTXO[txID]
				outs.Outputs = append(outs.Outputs, out)
				UTXO[txID] = outs
			}

			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					inTxID := hex.EncodeToString(in.ID)
					spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Out)
				}
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return UTXO
}

func (chain *BlockChain) FindTransaction(ID []byte) (Transaction, error) {
	iter := chain.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			if bytes.Equal(tx.ID, ID) {
				return tx, nil
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("Transaction does not exist")
}

func (chain *BlockChain) SignTransaction(tx Transaction, privKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTX, err := chain.FindTransaction(in.ID)
		utils.Handle(err)

		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	tx.Sign(&privKey, prevTXs)
}

func (chain *BlockChain) VerifyTransaction(tx Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	prevTXs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		prevTX, err := chain.FindTransaction(in.ID)
		utils.Handle(err)

		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs)
}
