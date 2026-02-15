package main

import (
	"os"

	"github.com/mkusaka/tfdc/internal/cli"
)

func main() {
	code := cli.Execute(os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(code)
}
