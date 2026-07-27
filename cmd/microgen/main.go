package main

import (
	"os"

	"github.com/pozeydon-code/microservices-generator-csharp/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
