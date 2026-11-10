package main

import (
	"fmt"
	"os"

	"github.com/rhyan05/gitcontrol/internal/cli"
	"github.com/rhyan05/gitcontrol/internal/git"
)

var version = "0.1.0"

const usage = `uso:
  gitcontrol [--repo <path>] [repo...]        abre a TUI
  gitcontrol --json [--repo <path>] <comando> contrato JSON
  gitcontrol --version                        mostra a versão
  gitcontrol --help                           mostra esta ajuda`

func main() {
	args := os.Args[1:]
	jsonMode := false
	repoPath := ""

parse:
	for len(args) > 0 {
		switch args[0] {
		case "--json":
			jsonMode = true
			args = args[1:]
		case "--repo":
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "--repo exige um caminho")
				os.Exit(2)
			}
			repoPath = args[1]
			args = args[2:]
		case "--version", "-v":
			fmt.Println("gitcontrol " + version)
			return
		case "--help", "-h":
			fmt.Println(usage)
			return
		default:
			break parse
		}
	}

	if jsonMode {
		path := repoPath
		if path == "" {
			cwd, err := os.Getwd()
			if err != nil {
				fmt.Fprintf(os.Stderr, "gitcontrol: %v\n", err)
				os.Exit(1)
			}
			path = cwd
		}
		os.Exit(cli.Run(git.New(path), args, os.Stdout))
	}

	repos := args
	if repoPath != "" {
		repos = append([]string{repoPath}, repos...)
	}
	if err := run(repos); err != nil {
		fmt.Fprintf(os.Stderr, "gitcontrol: %v\n", err)
		os.Exit(1)
	}
}
