package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"math/big"

	"github.com/dev-rodrigobaliza/go-blockchain/utils"
	"golang.org/x/crypto/ripemd160"
)

func NewKeyPair() (*ecdsa.PrivateKey, []byte, []byte) {
	curve := elliptic.P256()

	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	utils.Handle(err)

	pub := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	privateBytes := elliptic.Marshal(private.Curve, private.X, private.Y)

	return private, privateBytes, pub
}

func PrivateKeyFromBytes(privateBytes []byte) *ecdsa.PrivateKey {
	private := new(ecdsa.PrivateKey)
	private.Curve = elliptic.P256()
	private.D = new(big.Int).SetBytes(privateBytes)
	private.X, private.Y = elliptic.Unmarshal(elliptic.P256(), privateBytes)

	return private
}

func PublicKeyHash(pubKey []byte) []byte {
	pubHash := sha256.Sum256(pubKey)

	hasher := ripemd160.New()
	_, err := hasher.Write(pubHash[:])
	utils.Handle(err)

	publicRipMD := hasher.Sum(nil)

	return publicRipMD
}
