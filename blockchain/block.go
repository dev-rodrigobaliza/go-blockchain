package blockchain

import (
	"time"

	"github.com/dev-rodrigobaliza/go-blockchain/utils"
	"github.com/goccy/go-json"
)

type Block struct {
	Timestamp    int64
	Hash         []byte        `json:"hash,omitempty"`
	Transactions []Transaction `json:"transactions,omitempty"`
	PrevHash     []byte        `json:"prev_hash,omitempty"`
	Nonce        int
	Height       int
}

func NewBlock(txs []Transaction, prevHash []byte, height int) Block {
	block := Block{time.Now().Unix(), []byte{}, txs, prevHash, 0, height}
	pow := NewProof(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

func Genesis(coinbase Transaction) Block {
	return NewBlock([]Transaction{coinbase}, []byte{}, 0)
}

func (b Block) HashTransactions() []byte {
	var txHashes [][]byte

	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.Serialize())
	}
	tree := NewMerkleTree(txHashes)

	return tree.RootNode.Data
}

func (b Block) Serialize() []byte {
	buffer, err := json.Marshal(b)
	utils.Handle(err)

	return buffer
}

func (b Block) Deserialize(buffer []byte) error {
	err := json.Unmarshal(buffer, b)
	if err != nil {
		return err
	}

	return nil
}
