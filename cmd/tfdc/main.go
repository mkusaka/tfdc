package main

import (
	"os"

	"github.com/mkusaka/tfdc/internal/cli"
)

func main() {
	code := cli.Execute(os.Args[1:], os.Stderr)
	os.Exit(code)
}
