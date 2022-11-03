package blockchain

import (
	"bytes"

	"github.com/dev-rodrigobaliza/go-blockchain/crypto"
)

type TxInput struct {
	ID        []byte `json:"id,omitempty"`
	Out       int
	Signature []byte `json:"signature,omitempty"`
	PubKey    []byte `json:"pub_key,omitempty"`
}

func NewTxInput(id []byte, out int, signature []byte, pubKey []byte) TxInput {
	return TxInput{id, out, signature, pubKey}
}

func (in TxInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := crypto.PublicKeyHash(in.PubKey)

	return bytes.Equal(lockingHash, pubKeyHash)
}
