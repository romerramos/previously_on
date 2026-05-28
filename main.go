package main

import (
	"os"

	"github.com/romerramos/previously_on/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
