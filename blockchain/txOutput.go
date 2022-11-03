package blockchain

import (
	"bytes"

	"github.com/dev-rodrigobaliza/go-blockchain/base58"
	"github.com/dev-rodrigobaliza/go-blockchain/wallet"
)

type TxOutput struct {
	Value      int
	PubKeyHash []byte `json:"pub_key_hash,omitempty"`
}

func NewTxOutput(value int, address string) *TxOutput {
	txo := TxOutput{value, nil}
	txo.Lock([]byte(address))

	return &txo
}

func (out *TxOutput) Lock(address []byte) {
	pubKeyHash := base58.Decode(address)
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-wallet.ChecksumLength]
	out.PubKeyHash = pubKeyHash
}

func (out *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Equal(out.PubKeyHash, pubKeyHash)
}
