package main

import (
	"os"

	"github.com/adams100111/agentic-learning-partner/internal/cli"
)

func main() {
	os.Exit(cli.New().Run(os.Args[1:]))
}
