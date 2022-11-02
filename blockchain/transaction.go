package blockchain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/dev-rodrigobaliza/go-blockchain/crypto"
	"github.com/dev-rodrigobaliza/go-blockchain/utils"
	"github.com/dev-rodrigobaliza/go-blockchain/wallet"
	"github.com/goccy/go-json"
)

type Transaction struct {
	ID      []byte
	Inputs  []*TxInput
	Outputs []*TxOutput
}

func NewTransaction(from, to string, amount int, UTXO *UTXOSet) *Transaction {
	var inputs []*TxInput
	var outputs []*TxOutput

	wallets, err := wallet.NewWallets()
	utils.Handle(err)

	w := wallets.GetWallet(from)
	pubKeyHash := crypto.PublicKeyHash(w.PublicKey)

	acc, validOutputs := UTXO.FindSpendableOutputs(pubKeyHash, amount)
	if acc < amount {
		log.Panic("Error: not enough funds")
	}

	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		utils.Handle(err)

		for _, out := range outs {
			input := NewTxInput(txID, out, nil, w.PublicKey)
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, NewTxOutput(amount, []byte(to)))
	if acc > amount {
		outputs = append(outputs, NewTxOutput(acc-amount, []byte(from)))
	}

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()
	UTXO.Blockchain.SignTransaction(&tx, *w.GetPrivateKey())

	return &tx
}

func (tx *Transaction) Serialize() []byte {
	buffer, err := json.Marshal(tx)
	utils.Handle(err)

	return buffer
}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	tx.ID = []byte{}
	hash = sha256.Sum256(txCopy.Serialize())

	return hash[:]
}

func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	var hash [32]byte

	encode := gob.NewEncoder(&encoded)
	err := encode.Encode(tx)
	utils.Handle(err)

	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Out == -1
}

func (tx *Transaction) Sign(privKey *ecdsa.PrivateKey, prevTXs map[string]*Transaction) {
	if tx.IsCoinbase() {
		return
	}

	for _, in := range tx.Inputs {
		if prevTXs[hex.EncodeToString(in.ID)].ID == nil {
			log.Panic("Error: previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy()

	for inId, in := range txCopy.Inputs {
		prevTX := prevTXs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = prevTX.Outputs[in.Out].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inId].PubKey = nil

		r, s, err := ecdsa.Sign(rand.Reader, privKey, txCopy.ID)
		utils.Handle(err)
		signature := append(r.Bytes(), s.Bytes()...)

		tx.Inputs[inId].Signature = signature
	}
}

func (tx Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("--- Transaction %x:", tx.ID))
	for i, input := range tx.Inputs {

		lines = append(lines, fmt.Sprintf("     Input %d:", i))
		lines = append(lines, fmt.Sprintf("       TXID:      %x", input.ID))
		lines = append(lines, fmt.Sprintf("       Out:       %d", input.Out))
		lines = append(lines, fmt.Sprintf("       Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("       PubKey:    %x", input.PubKey))
	}

	for i, output := range tx.Outputs {
		lines = append(lines, fmt.Sprintf("     Output %d:", i))
		lines = append(lines, fmt.Sprintf("       Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("       Script: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}

func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []*TxInput
	var outputs []*TxOutput

	for _, input := range tx.Inputs {
		inputs = append(inputs, NewTxInput(input.ID, input.Out, nil, nil))
	}

	for _, output := range tx.Outputs {
		outputs = append(outputs, NewTxOutput(output.Value, output.PubKeyHash))
	}

	txCopy := Transaction{tx.ID, inputs, outputs}

	return txCopy
}

func (tx *Transaction) Verify(prevTXs map[string]*Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	for _, input := range tx.Inputs {
		if prevTXs[hex.EncodeToString(input.ID)].ID == nil {
			log.Panic("Previous transaction is not correct")
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inID, input := range tx.Inputs {
		prevTx := prevTXs[hex.EncodeToString(input.ID)]
		txCopy.Inputs[inID].Signature = nil
		txCopy.Inputs[inID].PubKey = prevTx.Outputs[input.Out].PubKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inID].PubKey = nil

		r := big.Int{}
		s := big.Int{}
		sigLen := len(input.Signature)
		r.SetBytes(input.Signature[:(sigLen / 2)])
		s.SetBytes(input.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}
		keyLen := len(input.PubKey)
		x.SetBytes(input.PubKey[:(keyLen / 2)])
		y.SetBytes(input.PubKey[(keyLen / 2):])

		rawPubKey := &ecdsa.PublicKey{
			Curve: curve,
			X:     &x,
			Y:     &y,
		}
		if !ecdsa.Verify(rawPubKey, txCopy.ID, &r, &s) {
			return false
		}
	}

	return true
}

func CoinbaseTx(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("coins to %s", to)
	}

	txIn := NewTxInput([]byte{}, -1, nil, []byte(data))
	txOut := NewTxOutput(100, []byte(to))

	tx := Transaction{nil, []*TxInput{txIn}, []*TxOutput{txOut}}
	tx.SetID()

	return &tx
}
