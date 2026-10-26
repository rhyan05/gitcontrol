// Package cli implementa o contrato máquina do gg (modo --json).
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

	"github.com/rhyan05/gerenciador-git/internal/git"
)

const usage = "uso: gg --json <status|log|branches|show|checkout|stage|unstage|commit>"

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
			return fail(out, "usage", "uso: gg --json show <hash>")
		}
		diff, err := repo.Show(ctx, rest[0])
		if err != nil {
			return failGit(out, err)
		}
		return ok(out, showData{Hash: rest[0], Diff: diff})

	case "checkout":
		if len(rest) != 1 {
			return fail(out, "usage", "uso: gg --json checkout <branch>")
		}
		if err := repo.Checkout(ctx, rest[0]); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "stage":
		if len(rest) == 0 {
			return fail(out, "usage", "uso: gg --json stage <path...>")
		}
		if err := repo.Stage(ctx, rest); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "unstage":
		if len(rest) == 0 {
			return fail(out, "usage", "uso: gg --json unstage <path...>")
		}
		if err := repo.Unstage(ctx, rest); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "commit":
		if len(rest) == 0 {
			return fail(out, "usage", "uso: gg --json commit <mensagem...>")
		}
		// Junta os args para permitir `gg --json commit primeira mensagem`.
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
