package blockchain

import (
	"bytes"

	"github.com/dev-rodrigobaliza/go-blockchain/base58"
	"github.com/dev-rodrigobaliza/go-blockchain/crypto"
	"github.com/dev-rodrigobaliza/go-blockchain/wallet"
)

type TxOutput struct {
	Value      int
	PubKeyHash []byte
}

func NewTxOutput(value int, address string) *TxOutput {
	txo := TxOutput{value, nil}
	txo.Lock([]byte(address))

	return &txo
}

type TxInput struct {
	ID        []byte
	Out       int
	Signature []byte
	PubKey    []byte
}

func (in *TxInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := crypto.PublicKeyHash(in.PubKey)

	return bytes.Equal(lockingHash, pubKeyHash)
}

func (out *TxOutput) Lock(address []byte) {
	pubKeyHash := base58.Decode(address)
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-wallet.ChecksumLength]
	out.PubKeyHash = pubKeyHash
}

func (out *TxOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Equal(out.PubKeyHash, pubKeyHash)
}
