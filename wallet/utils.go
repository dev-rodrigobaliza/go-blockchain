package wallet

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dev-rodrigobaliza/go-blockchain/base58"
	"github.com/dev-rodrigobaliza/go-blockchain/utils"
)

const (
	walletsPath = "wallets"
	walletsFile = "wallets_%s.data"
)

func checksum(payload []byte) []byte {
	firstHash := sha256.Sum256(payload)
	secondHash := sha256.Sum256(firstHash[:])

	return secondHash[:ChecksumLength]
}

func checkWalletsPath(nodeId string) string {
	systemPath := utils.CheckSystemPath()
	wsPath := filepath.Join(systemPath, walletsPath)
	_ = os.Mkdir(wsPath, os.ModePerm)

	wFile := fmt.Sprintf(walletsFile, nodeId)
	wPath := filepath.Join(wsPath, wFile)

	return wPath
}

func ValidateAddress(address string) bool {
	pubKeyHash := base58.Decode([]byte(address))
	sourceChecksum := pubKeyHash[len(pubKeyHash)-ChecksumLength:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-ChecksumLength]
	targetChecksum := checksum(append([]byte{version}, pubKeyHash...))

	return bytes.Equal(sourceChecksum, targetChecksum)
}
