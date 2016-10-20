package main

import (
	"fmt"
	"os"

	"github.com/vcabbage/unpkg/cmd/unpkg/cli"
	"github.com/vcabbage/unpkg/cmd/unpkg/server"
)

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) < 2 {
		fmt.Println(`must specify "serve" or package`)
		return 1
	}

	if os.Args[1] == "serve" {
		return server.Run()
	}

	return cli.Run()
}
