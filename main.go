package main

import (
	"os"

	"github.com/dev-rodrigobaliza/go-blockchain/cli"
)

func main() {
	defer os.Exit(0)
	cli := cli.CommandLine{}
	cli.Run()
}
