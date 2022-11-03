package blockchain

import (
	"github.com/dev-rodrigobaliza/go-blockchain/utils"
	"github.com/goccy/go-json"
)

type TxOutputs struct {
	Outputs []TxOutput
}

func (t *TxOutputs) serialize() []byte {
	buffer, err := json.Marshal(t)
	utils.Handle(err)

	return buffer
}

func (t *TxOutputs) deserialize(buffer []byte) error {
	err := json.Unmarshal(buffer, t)
	if err != nil {
		return err
	}

	return nil
}
