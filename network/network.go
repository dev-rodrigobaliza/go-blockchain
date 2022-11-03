package network

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"syscall"

	"github.com/dev-rodrigobaliza/go-blockchain/blockchain"
	"github.com/dev-rodrigobaliza/go-blockchain/utils"
	"github.com/vrecan/death/v3"
)

const (
	protocol      = "tcp"
	version       = 1
	commandLength = 12
)

var (
	nodeAddress     string
	miningAddress   string
	KnownNodes      = []string{"localhost:3000"}
	blocksInTransit = [][]byte{}
	memoryPool      = make(map[string]blockchain.Transaction)
)

type addr struct {
	AddrList []string
}

type block struct {
	AddrFrom string
	Block    []byte
}

type getBlocks struct {
	AddrFrom string
}

type getData struct {
	AddrFrom string
	Type     string
	ID       []byte
}

type inv struct {
	AddrFrom string
	Type     string
	Items    [][]byte
}

type tx struct {
	AddrFrom    string
	Transaction []byte
}

type Version struct {
	Version    int
	BestHeight int
	AddrFrom   string
}

func StartServer(nodeID, minerAddress string) {
	nodeAddress = fmt.Sprintf("localhost:%s", nodeID)
	miningAddress = minerAddress

	ln, err := net.Listen(protocol, nodeAddress)
	utils.Handle(err)
	defer ln.Close()

	chain := blockchain.ContinueBlockChain(nodeID)
	defer chain.Database.Close()
	go closeDB(chain)

	if nodeAddress != KnownNodes[0] {
		sendVersion(KnownNodes[0], chain)
	}

	for {
		conn, err := ln.Accept()
		utils.Handle(err)

		go handleConnection(conn, chain)
	}
}

func serialize(cmd string) []byte {
	var bytes [commandLength]byte

	for i, c := range cmd {
		bytes[i] = byte(c)
	}

	return bytes[:]
}

func deserialize(bytes []byte) string {
	var cmd []byte

	for _, b := range bytes {
		if b != 0x00 {
			cmd = append(cmd, b)
		}
	}

	return string(cmd)
}

func requestBlocks() {
	for _, node := range KnownNodes {
		sendGetBlocks(node)
	}
}

func closeDB(chain *blockchain.BlockChain) {
	d := death.NewDeath(syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	d.WaitForDeathWithFunc(func() {
		defer os.Exit(1)
		defer runtime.Goexit()
		chain.Database.Close()
	})
}

func gobEncode(data interface{}) []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	utils.Handle(err)

	return buff.Bytes()
}

func handleConnection(conn net.Conn, chain *blockchain.BlockChain) {
	request, err := io.ReadAll(conn)
	utils.Handle(err)
	defer conn.Close()

	command := deserialize(request[:commandLength])
	fmt.Printf("Received %s command\n", command)

	switch command {
	case "addr":
		handleAddr(request)

	case "block":
		handleBlock(request, chain)

	case "inv":
		handleInv(request, chain)

	case "getblocks":
		handleGetBlocks(request, chain)

	case "getdata":
		handleGetData(request, chain)

	case "tx":
		handleTx(request, chain)

	case "version":
		handleVersion(request, chain)

	default:
		fmt.Println("Unknown command!")
	}
}

func handleAddr(request []byte) {
	var buff bytes.Buffer
	var payload addr

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	utils.Handle(err)

	KnownNodes = append(KnownNodes, payload.AddrList...)
	fmt.Printf("There are %d known nodes now!\n", len(KnownNodes))
	requestBlocks()
}

func handleBlock(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload block

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	utils.Handle(err)

	blockData := payload.Block
	var block *blockchain.Block
	err = block.Deserialize(blockData)
	utils.Handle(err)

	fmt.Println("Recevied a new block!")
	chain.AddBlock(block)

	fmt.Printf("Added block %x\n", block.Hash)

	if len(blocksInTransit) > 0 {
		blockHash := blocksInTransit[0]
		sendGetData(payload.AddrFrom, "block", blockHash)

		blocksInTransit = blocksInTransit[1:]
	} else {
		UTXOSet := blockchain.UTXOSet{
			Blockchain: chain,
		}
		UTXOSet.Reindex()
	}
}

func handleInv(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload inv

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	utils.Handle(err)

	fmt.Printf("Recevied inventory with %d %s\n", len(payload.Items), payload.Type)

	switch payload.Type {
	case "block":
		blocksInTransit = payload.Items

		blockHash := payload.Items[0]
		sendGetData(payload.AddrFrom, "block", blockHash)

		newInTransit := [][]byte{}
		for _, b := range blocksInTransit {
			if !bytes.Equal(b, blockHash) {
				newInTransit = append(newInTransit, b)
			}
		}
		blocksInTransit = newInTransit

	case "tx":
		txID := payload.Items[0]

		if memoryPool[hex.EncodeToString(txID)].ID == nil {
			sendGetData(payload.AddrFrom, "tx", txID)
		}
	}
}

func handleGetBlocks(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload getBlocks

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	utils.Handle(err)

	blocks := chain.GetBlockHashes()
	sendInv(payload.AddrFrom, "block", blocks)
}

func handleGetData(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload getData

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	utils.Handle(err)

	if payload.Type == "block" {
		block, err := chain.GetBlock([]byte(payload.ID))
		if err != nil {
			return
		}

		sendBlock(payload.AddrFrom, block)
	}

	if payload.Type == "tx" {
		txID := hex.EncodeToString(payload.ID)
		tx := memoryPool[txID]

		SendTx(payload.AddrFrom, tx)
		// delete(mempool, txID)
	}
}

func handleTx(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload tx

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	utils.Handle(err)

	txData := payload.Transaction
	var tx blockchain.Transaction
	err = tx.Deserialize(txData)
	utils.Handle(err)

	memoryPool[hex.EncodeToString(tx.ID)] = tx

	fmt.Printf("%s, %d", nodeAddress, len(memoryPool))

	if nodeAddress == KnownNodes[0] {
		for _, node := range KnownNodes {
			if node != nodeAddress && node != payload.AddrFrom {
				sendInv(node, "tx", [][]byte{tx.ID})
			}
		}
	} else {
		if len(memoryPool) >= 2 && len(miningAddress) > 0 {
			mineTx(chain)
		}
	}
}

func handleVersion(request []byte, chain *blockchain.BlockChain) {
	var buff bytes.Buffer
	var payload Version

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	utils.Handle(err)

	myBestHeight := chain.GetBestHeight()
	foreignerBestHeight := payload.BestHeight

	if myBestHeight < foreignerBestHeight {
		sendGetBlocks(payload.AddrFrom)
	} else if myBestHeight > foreignerBestHeight {
		sendVersion(payload.AddrFrom, chain)
	}

	if !nodeIsKnown(payload.AddrFrom) {
		KnownNodes = append(KnownNodes, payload.AddrFrom)
	}
}

func sendBlock(addr string, b *blockchain.Block) {
	data := block{nodeAddress, b.Serialize()}
	payload := gobEncode(data)
	request := append(serialize("block"), payload...)

	sendData(addr, request)
}

func sendData(addr string, data []byte) {
	conn, err := net.Dial(protocol, addr)
	if err != nil {
		fmt.Printf("%s is not available\n", addr)
		var updatedNodes []string

		for _, node := range KnownNodes {
			if node != addr {
				updatedNodes = append(updatedNodes, node)
			}
		}

		KnownNodes = updatedNodes

		return
	}
	defer conn.Close()

	_, err = io.Copy(conn, bytes.NewReader(data))
	utils.Handle(err)
}

func sendGetBlocks(address string) {
	payload := gobEncode(getBlocks{nodeAddress})
	request := append(serialize("getblocks"), payload...)

	sendData(address, request)
}

func sendGetData(address, kind string, id []byte) {
	payload := gobEncode(getData{nodeAddress, kind, id})
	request := append(serialize("getdata"), payload...)

	sendData(address, request)
}

func sendInv(address, kind string, items [][]byte) {
	inventory := inv{nodeAddress, kind, items}
	payload := gobEncode(inventory)
	request := append(serialize("inv"), payload...)

	sendData(address, request)
}

func SendTx(addr string, tnx blockchain.Transaction) {
	data := tx{nodeAddress, tnx.Serialize()}
	payload := gobEncode(data)
	request := append(serialize("tx"), payload...)

	sendData(addr, request)
}

func sendVersion(addr string, bc *blockchain.BlockChain) {
	bestHeight := bc.GetBestHeight()
	payload := gobEncode(Version{version, bestHeight, nodeAddress})

	request := append(serialize("version"), payload...)

	sendData(addr, request)
}

func nodeIsKnown(addr string) bool {
	for _, node := range KnownNodes {
		if node == addr {
			return true
		}
	}

	return false
}

func mineTx(chain *blockchain.BlockChain) {
	var txs []blockchain.Transaction

	for id := range memoryPool {
		fmt.Printf("tx: %s\n", memoryPool[id].ID)
		tx := memoryPool[id]
		if chain.VerifyTransaction(tx) {
			txs = append(txs, tx)
		}
	}

	if len(txs) == 0 {
		fmt.Println("All transactions are invalid")
		return
	}

	cbTx := blockchain.CoinbaseTx(miningAddress, "")
	txs = append(txs, cbTx)

	newBlock := chain.MineBlock(txs)
	UTXOSet := blockchain.UTXOSet{
		Blockchain: chain,
	}
	UTXOSet.Reindex()

	fmt.Println("New block mined")

	for _, tx := range txs {
		txID := hex.EncodeToString(tx.ID)
		delete(memoryPool, txID)
	}

	for _, node := range KnownNodes {
		if node != nodeAddress {
			sendInv(node, "block", [][]byte{newBlock.Hash})
		}
	}

	if len(memoryPool) > 0 {
		mineTx(chain)
	}
}
