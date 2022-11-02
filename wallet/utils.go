package wallet

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"

	"github.com/dev-rodrigobaliza/go-blockchain/base58"
	"github.com/dev-rodrigobaliza/go-blockchain/utils"
)

func checksum(payload []byte) []byte {
	firstHash := sha256.Sum256(payload)
	secondHash := sha256.Sum256(firstHash[:])

	return secondHash[:ChecksumLength]
}

func file() string {
	curPath, err := os.Getwd()
	utils.Handle(err)

	return filepath.Join(curPath, walletFile)
}

func ValidateAddress(address string) bool {
	pubKeyHash := base58.Decode([]byte(address))
	sourceChecksum := pubKeyHash[len(pubKeyHash)-ChecksumLength:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-ChecksumLength]
	targetChecksum := checksum(append([]byte{version}, pubKeyHash...))

	return bytes.Equal(sourceChecksum, targetChecksum)
}
