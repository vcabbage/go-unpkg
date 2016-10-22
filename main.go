package main

import (
	"os"

	"github.com/vcabbage/go-unpkg/server"
)

func main() {
	os.Exit(server.Run())
}
