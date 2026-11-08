package main

import (
	"github.com/alzubair-alojali/hova/internal/cli"
	"os"
)

var version = "dev"

func main() { os.Exit(cli.Execute(os.Args[1:], os.Stdout, os.Stderr, version)) }
