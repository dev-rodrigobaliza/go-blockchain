package wallet

import (
	"crypto/ecdsa"
)

const (
	checksumLength = 4
	version        = byte(0x00)
)

type Wallet struct {
	privateKey *ecdsa.PrivateKey
	PrivateKey []byte `json:"private_key"`
	PublicKey  []byte `json:"public_key"`
}

func NewWallet() *Wallet {
	private, privateBytes, public := newKeyPair()
	wallet := Wallet{private, privateBytes, public}

	return &wallet
}

func (w *Wallet) Address() []byte {
	pubHash := publicKeyHash(w.PublicKey)
	versionedHash := append([]byte{version}, pubHash...)
	checksum := checksum(versionedHash)
	fullHash := append(versionedHash, checksum...)
	address := base58Encode(fullHash)

	// fmt.Printf("pub key: %x\n", w.PublicKey)
	// fmt.Printf("pub hash: %x\n", pubHash)
	// fmt.Printf("address: %s\n", address)

	return address
}
