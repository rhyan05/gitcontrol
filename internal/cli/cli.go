// Package cli implementa o contrato máquina do gitcontrol (modo --json).
//
// Toda resposta é um único objeto JSON no stdout:
//
//	{"ok":true,"data":...}
//	{"ok":false,"error":{"code":"...","message":"..."}}
//
// O exit code é 0 em sucesso e 1 em erro. O Tauri (Bloco 9) consome
// exatamente esse contrato.
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/rhyan05/gitcontrol/internal/git"
)

const usage = "uso: gitcontrol --json <status|log|graph|branches|show|diff|remotes|fetch|pull|push|checkout|merge|cherry-pick|resolve|abort|continue|stage|unstage|commit>"

// success e failure são structs separados para o JSON de sucesso sempre ter
// "data" (mesmo que null) e o de erro sempre ter "error".
type success struct {
	OK   bool `json:"ok"`
	Data any  `json:"data"`
}

type failure struct {
	OK    bool      `json:"ok"`
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// showData é o data de `show <hash>`: o hash pedido + o diff completo.
type showData struct {
	Hash string `json:"hash"`
	Diff string `json:"diff"`
}

// diffData é o data de `diff <path>`: o path pedido + o diff do working tree.
type diffData struct {
	Path string `json:"path"`
	Diff string `json:"diff"`
}

// Run executa um comando do contrato e escreve o JSON em out.
// Retorna o exit code (0 sucesso, 1 erro); o chamador só faz os.Exit.
func Run(repo *git.Repo, args []string, out io.Writer) int {
	if len(args) == 0 {
		return fail(out, "usage", usage)
	}

	cmd, rest := args[0], args[1:]
	ctx := context.Background()

	switch cmd {
	case "status":
		if len(rest) != 0 {
			return fail(out, "usage", "status não aceita argumentos")
		}
		st, err := repo.Status(ctx)
		if err != nil {
			return failGit(out, err)
		}
		return ok(out, st)

	case "log":
		if len(rest) != 0 {
			return fail(out, "usage", "log não aceita argumentos")
		}
		commits, err := repo.Log(ctx, 50)
		if err != nil {
			return failGit(out, err)
		}
		return ok(out, commits)

	case "graph":
		if len(rest) != 0 {
			return fail(out, "usage", "graph não aceita argumentos")
		}
		rows, err := repo.Graph(ctx, 50)
		if err != nil {
			return failGit(out, err)
		}
		return ok(out, rows)

	case "branches":
		if len(rest) != 0 {
			return fail(out, "usage", "branches não aceita argumentos")
		}
		branches, err := repo.Branches(ctx)
		if err != nil {
			return failGit(out, err)
		}
		return ok(out, branches)

	case "show":
		if len(rest) != 1 {
			return fail(out, "usage", "uso: gitcontrol --json show <hash>")
		}
		diff, err := repo.Show(ctx, rest[0])
		if err != nil {
			return failGit(out, err)
		}
		return ok(out, showData{Hash: rest[0], Diff: diff})

	case "diff":
		if len(rest) != 1 {
			return fail(out, "usage", "uso: gitcontrol --json diff <path>")
		}
		diff, err := repo.DiffFile(ctx, rest[0])
		if err != nil {
			return failGit(out, err)
		}
		return ok(out, diffData{Path: rest[0], Diff: diff})

	case "remotes":
		if len(rest) != 0 {
			return fail(out, "usage", "remotes não aceita argumentos")
		}
		remotes, err := repo.Remotes(ctx)
		if err != nil {
			return failGit(out, err)
		}
		return ok(out, remotes)

	case "fetch":
		if len(rest) > 1 {
			return fail(out, "usage", "uso: gitcontrol --json fetch [remote]")
		}
		remote := ""
		if len(rest) == 1 {
			remote = rest[0]
		}
		if err := repo.Fetch(ctx, remote); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "pull":
		if len(rest) > 2 {
			return fail(out, "usage", "uso: gitcontrol --json pull [remote] [branch]")
		}
		remote, branch := "", ""
		if len(rest) >= 1 {
			remote = rest[0]
		}
		if len(rest) == 2 {
			branch = rest[1]
		}
		if err := repo.Pull(ctx, remote, branch); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "push":
		if len(rest) > 2 {
			return fail(out, "usage", "uso: gitcontrol --json push [remote] [branch]")
		}
		remote, branch := "", ""
		if len(rest) >= 1 {
			remote = rest[0]
		}
		if len(rest) == 2 {
			branch = rest[1]
		}
		if err := repo.Push(ctx, remote, branch); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "checkout":
		if len(rest) != 1 {
			return fail(out, "usage", "uso: gitcontrol --json checkout <branch>")
		}
		if err := repo.Checkout(ctx, rest[0]); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "merge":
		if len(rest) != 1 {
			return fail(out, "usage", "uso: gitcontrol --json merge <branch>")
		}
		if err := repo.Merge(ctx, rest[0]); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "cherry-pick":
		if len(rest) != 1 {
			return fail(out, "usage", "uso: gitcontrol --json cherry-pick <hash>")
		}
		if err := repo.CherryPick(ctx, rest[0]); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "resolve":
		if len(rest) == 0 {
			return fail(out, "usage", "uso: gitcontrol --json resolve <path...>")
		}
		if err := repo.Resolve(ctx, rest); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "abort":
		if len(rest) != 0 {
			return fail(out, "usage", "abort não aceita argumentos")
		}
		if err := repo.Abort(ctx); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "continue":
		if len(rest) != 0 {
			return fail(out, "usage", "continue não aceita argumentos")
		}
		if err := repo.Continue(ctx); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "stage":
		if len(rest) == 0 {
			return fail(out, "usage", "uso: gitcontrol --json stage <path...>")
		}
		if err := repo.Stage(ctx, rest); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "unstage":
		if len(rest) == 0 {
			return fail(out, "usage", "uso: gitcontrol --json unstage <path...>")
		}
		if err := repo.Unstage(ctx, rest); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "commit":
		if len(rest) == 0 {
			return fail(out, "usage", "uso: gitcontrol --json commit <mensagem...>")
		}
		// Junta os args para permitir `gitcontrol --json commit primeira mensagem`.
		if err := repo.Commit(ctx, strings.Join(rest, " ")); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	default:
		return fail(out, "usage", fmt.Sprintf("comando desconhecido: %s", cmd))
	}
}

// ok escreve o envelope de sucesso. data pode ser nil (data:null).
func ok(out io.Writer, data any) int {
	return encode(out, success{OK: true, Data: data}, 0)
}

// fail escreve o envelope de erro com o código informado.
func fail(out io.Writer, code, message string) int {
	return encode(out, failure{
		OK:    false,
		Error: errorBody{Code: code, Message: message},
	}, 1)
}

// failGit diferencia "não é um repositório" de outros erros do git, para o
// consumidor poder mostrar uma mensagem melhor.
func failGit(out io.Writer, err error) int {
	code := "git_error"
	if strings.Contains(err.Error(), "not a git repository") {
		code = "not_a_repo"
	}
	return fail(out, code, err.Error())
}

// encode centraliza o encoder: um objeto por execução, sem escapar HTML
// (paths com &, <, > ficam legíveis) e com newline no fim.
func encode(out io.Writer, value any, exit int) int {
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(value); err != nil {
		return 1
	}
	return exit
}
