package wallet

import (
	"crypto/ecdsa"

	"github.com/dev-rodrigobaliza/go-blockchain/base58"
	"github.com/dev-rodrigobaliza/go-blockchain/crypto"
)

const (
	ChecksumLength = 4
	version        = byte(0x00)
)

type Wallet struct {
	privateKey *ecdsa.PrivateKey
	PrivateKey []byte `json:"private_key"`
	PublicKey  []byte `json:"public_key"`
}

func NewWallet() *Wallet {
	private, privateBytes, public := crypto.NewKeyPair()
	wallet := Wallet{private, privateBytes, public}

	return &wallet
}

func (w *Wallet) Address() []byte {
	pubHash := crypto.PublicKeyHash(w.PublicKey)
	versionedHash := append([]byte{version}, pubHash...)
	checksum := checksum(versionedHash)
	fullHash := append(versionedHash, checksum...)
	address := base58.Encode(fullHash)

	return address
}

func (w *Wallet) GetPrivateKey() *ecdsa.PrivateKey {
	return w.privateKey
}
