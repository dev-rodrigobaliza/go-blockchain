package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"

	"github.com/dev-rodrigobaliza/go-blockchain/base58"
	"github.com/dev-rodrigobaliza/go-blockchain/blockchain"
	"github.com/dev-rodrigobaliza/go-blockchain/utils"
	"github.com/dev-rodrigobaliza/go-blockchain/wallet"
)

type CommandLine struct{}

func (cli *CommandLine) printUsage() {
	fmt.Println("Usage:")
	fmt.Println(" getbalance -address ADDRESS - get the balance for an address")
	fmt.Println(" createblockchain -address ADDRESS - create the blockchain for the given address")
	fmt.Println(" printchain - prints the blocks in the chain")
	fmt.Println(" send -from FROM -to TO -amount AMOUNT - send amount from one address to another address")
	fmt.Println(" createwallet - creates a new Wallet")
	fmt.Println(" listaddresses - lists the addresses in the wallet file")
	fmt.Println(" reindexutxo - rebuilds the UTXO set")
}

func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit()
	}
}

func (cli *CommandLine) reindexUTXO() {
	chain := blockchain.ContinueBlockChain("")
	defer chain.Database.Close()

	UTXOSet := blockchain.UTXOSet{
		Blockchain: chain,
	}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	fmt.Printf("Done, there are %d transactions in the UTXO set.\n", count)
}

func (cli *CommandLine) listAddresses() {
	wallets, err := wallet.NewWallets()
	utils.Handle(err)
	addresses := wallets.GetAllAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}

func (cli *CommandLine) createWallet() {
	wallets, err := wallet.NewWallets()
	utils.Handle(err)
	address := wallets.AddWallet()
	wallets.SaveFile()

	fmt.Printf("New address: %s\n", address)
}

func (cli *CommandLine) printChain() {
	chain := blockchain.ContinueBlockChain("")
	defer chain.Database.Close()

	iter := chain.Iterator()

	for {
		block := iter.Next()

		fmt.Printf("Hash: %x\n", block.Hash)
		fmt.Printf("Previous Hash: %x\n", block.PrevHash)
		pow := blockchain.NewProof(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}
		fmt.Println()

		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) createBlockChain(address string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is not valid")
	}

	chain := blockchain.InitBlockChain(address)
	defer chain.Database.Close()

	UTXOSet := blockchain.UTXOSet{
		Blockchain: chain,
	}
	UTXOSet.Reindex()

	fmt.Println("Finished")
}

func (cli *CommandLine) getBalance(address string) {
	if !wallet.ValidateAddress(address) {
		log.Panic("Address is not valid")
	}

	chain := blockchain.ContinueBlockChain(address)
	defer chain.Database.Close()

	UTXOSet := blockchain.UTXOSet{
		Blockchain: chain,
	}
	balance := 0
	pubKeyHash := base58.Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-wallet.ChecksumLength]
	UTXOs := UTXOSet.FindUnspentTransactions(pubKeyHash)

	for _, out := range UTXOs {
		balance += out.Value
	}

	fmt.Printf("Balance of %s: %d\n", address, balance)
}

func (cli *CommandLine) send(from, to string, amount int) {
	if !wallet.ValidateAddress(from) {
		log.Panic("From address is not valid")
	}

	if !wallet.ValidateAddress(to) {
		log.Panic("To address is not valid")
	}

	chain := blockchain.ContinueBlockChain(from)
	defer chain.Database.Close()

	UTXOSet := &blockchain.UTXOSet{
		Blockchain: chain,
	}
	tx := blockchain.NewTransaction(from, to, amount, UTXOSet)
	block := chain.AddBlock([]*blockchain.Transaction{tx})
	UTXOSet.Update(block)
	fmt.Println("Success")
}

func (cli *CommandLine) Run() {
	cli.validateArgs()

	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("print", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	reindexUTXOCmd := flag.NewFlagSet("reindexutxo", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "The address of the account")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address of the account")
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to sendt")

	switch os.Args[1] {
	case "reindexutxo":
		err := reindexUTXOCmd.Parse(os.Args[2:])
		utils.Handle(err)

	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		utils.Handle(err)

	case "listaddresses":
		err := listAddressesCmd.Parse(os.Args[2:])
		utils.Handle(err)

	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		utils.Handle(err)

	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		utils.Handle(err)

	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		utils.Handle(err)

	case "send":
		err := sendCmd.Parse(os.Args[2:])
		utils.Handle(err)

	default:
		cli.printUsage()
		runtime.Goexit()
	}

	if reindexUTXOCmd.Parsed() {
		cli.reindexUTXO()
	}

	if createWalletCmd.Parsed() {
		cli.createWallet()
	}

	if listAddressesCmd.Parsed() {
		cli.listAddresses()
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}
		cli.getBalance(*getBalanceAddress)
	}

	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			runtime.Goexit()
		}
		cli.createBlockChain(*createBlockchainAddress)
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			runtime.Goexit()
		}
		cli.send(*sendFrom, *sendTo, *sendAmount)
	}

	if printChainCmd.Parsed() {
		cli.printChain()
	}
}
