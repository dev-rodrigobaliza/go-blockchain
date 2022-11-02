package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"os"
	"path/filepath"

	"github.com/dev-rodrigobaliza/go-blockchain/utils"
	"github.com/mr-tron/base58"
	"golang.org/x/crypto/ripemd160"
)

func base58Encode(input []byte) []byte {
	encode := base58.Encode(input)

	return []byte(encode)
}

func base58Dencode(input []byte) []byte {
	decode, err := base58.Decode(string(input[:]))
	utils.Handle(err)

	return decode
}

func newKeyPair() (*ecdsa.PrivateKey, []byte, []byte) {
	curve := elliptic.P256()

	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	utils.Handle(err)

	pub := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	privateBytes := elliptic.Marshal(private.Curve, private.X, private.Y)

	return private, privateBytes, pub
}

func privateKeyFromBytes(privateBytes []byte) *ecdsa.PrivateKey {
	private := new(ecdsa.PrivateKey)
	private.Curve = elliptic.P256()
	private.X, private.Y = elliptic.Unmarshal(elliptic.P256(), privateBytes)

	return private
}

func publicKeyHash(pubKey []byte) []byte {
	pubHash := sha256.Sum256(pubKey)

	hasher := ripemd160.New()
	_, err := hasher.Write(pubHash[:])
	utils.Handle(err)

	publicRipMD := hasher.Sum(nil)

	return publicRipMD
}

func checksum(payload []byte) []byte {
	firstHash := sha256.Sum256(payload)
	secondHash := sha256.Sum256(firstHash[:])

	return secondHash[:checksumLength]
}

func file() string {
	curPath, err := os.Getwd()
	utils.Handle(err)

	return filepath.Join(curPath, walletFile)
}
