package main

import (
	"os"

	"github.com/mkusaka/terraform-docs-cli/internal/cli"
)

func main() {
	code := cli.Execute(os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(code)
}
