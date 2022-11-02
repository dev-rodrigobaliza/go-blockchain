package wallet

import (
	"os"

	"github.com/dev-rodrigobaliza/go-blockchain/crypto"
	"github.com/dev-rodrigobaliza/go-blockchain/utils"
	"github.com/goccy/go-json"
)

const (
	walletFile = "db/wallets.data"
)

type Wallets struct {
	Wallets map[string]*Wallet `json:"wallets"`
}

func NewWallets() (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.LoadFile()

	return &wallets, err
}

func (ws *Wallets) AddWallet() string {
	wallet := NewWallet()
	address := string(wallet.Address())

	ws.Wallets[address] = wallet

	return address
}

func (ws *Wallets) GetAllAddresses() []string {
	var addresses []string

	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

func (ws *Wallets) GetWallet(address string) *Wallet {
	return ws.Wallets[address]
}

func (ws *Wallets) LoadFile() error {
	_, err := os.Stat(file())
	if os.IsNotExist(err) {
		return nil
	}

	var wallets Wallets

	data, err := os.ReadFile(file())
	if err != nil {
		return err
	}

	err = wallets.fromBytes(data)
	if err != nil {
		return err
	}

	ws.Wallets = wallets.Wallets

	return nil
}

func (ws *Wallets) SaveFile() {
	data := ws.toBytes()

	err := os.WriteFile(file(), data, 0644)
	utils.Handle(err)
}

func (ws *Wallets) fromBytes(buffer []byte) error {
	err := json.Unmarshal(buffer, ws)
	if err != nil {
		return err
	}

	for address := range ws.Wallets {
		wallet := ws.Wallets[address]
		wallet.privateKey = crypto.PrivateKeyFromBytes(wallet.PrivateKey)
	}

	return nil
}

func (ws *Wallets) toBytes() []byte {
	buffer, err := json.Marshal(ws)
	utils.Handle(err)

	return buffer
}
