package base58

import (
	"github.com/dev-rodrigobaliza/go-blockchain/utils"
	"github.com/mr-tron/base58"
)

func Encode(input []byte) []byte {
	encode := base58.Encode(input)

	return []byte(encode)
}

func Decode(input []byte) []byte {
	decode, err := base58.Decode(string(input[:]))
	utils.Handle(err)

	return decode
}