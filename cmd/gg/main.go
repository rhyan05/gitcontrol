package main

import (
	"fmt"
	"os"

	"github.com/rhyan05/gerenciador-git/internal/cli"
	"github.com/rhyan05/gerenciador-git/internal/git"
)

func main() {
	// gg --json <comando> ... → contrato máquina.
	// Qualquer outra invocação abre a TUI.
	if len(os.Args) > 1 && os.Args[1] == "--json" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "gg: %v\n", err)
			os.Exit(1)
		}
		os.Exit(cli.Run(git.New(cwd), os.Args[2:], os.Stdout))
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "gg: %v\n", err)
		os.Exit(1)
	}
}
