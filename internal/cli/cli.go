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
	"strconv"
	"strings"

	"github.com/rhyan05/gitcontrol/internal/git"
)

const usage = "uso: gitcontrol --json <status|log|graph|branches|branch-create|branch-delete|branch-rename|show|diff|remotes|remote-add|remote-remove|fetch|pull|push|checkout|merge|cherry-pick|rebase|resolve|abort|continue|stage|unstage|discard|commit|amend|reset|stash|stash-list|stash-pop|stash-apply|stash-drop|tags|tag-create|tag-delete>"

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

	case "branch-create":
		if len(rest) < 1 || len(rest) > 2 {
			return fail(out, "usage", "uso: gitcontrol --json branch-create <name> [start]")
		}
		start := ""
		if len(rest) == 2 {
			start = rest[1]
		}
		if err := repo.CreateBranch(ctx, rest[0], start); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "branch-delete":
		force := false
		args := rest
		if len(args) > 0 && args[0] == "--force" {
			force = true
			args = args[1:]
		}
		if len(args) != 1 {
			return fail(out, "usage", "uso: gitcontrol --json branch-delete <name> [--force]")
		}
		if err := repo.DeleteBranch(ctx, args[0], force); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "branch-rename":
		if len(rest) != 2 {
			return fail(out, "usage", "uso: gitcontrol --json branch-rename <old> <new>")
		}
		if err := repo.RenameBranch(ctx, rest[0], rest[1]); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

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

	case "remote-add":
		if len(rest) != 2 {
			return fail(out, "usage", "uso: gitcontrol --json remote-add <name> <url>")
		}
		if err := repo.AddRemote(ctx, rest[0], rest[1]); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "remote-remove":
		if len(rest) != 1 {
			return fail(out, "usage", "uso: gitcontrol --json remote-remove <name>")
		}
		if err := repo.RemoveRemote(ctx, rest[0]); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

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

	case "rebase":
		if len(rest) != 1 {
			return fail(out, "usage", "uso: gitcontrol --json rebase <upstream>")
		}
		if err := repo.Rebase(ctx, rest[0]); err != nil {
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

	case "discard":
		if len(rest) == 0 {
			return fail(out, "usage", "uso: gitcontrol --json discard <path...>")
		}
		if err := repo.Discard(ctx, rest); err != nil {
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

	case "amend":
		if err := repo.Amend(ctx, strings.Join(rest, " ")); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "reset":
		if len(rest) < 1 || len(rest) > 2 {
			return fail(out, "usage", "uso: gitcontrol --json reset <rev> [soft|mixed|hard]")
		}
		mode := ""
		if len(rest) == 2 {
			mode = rest[1]
		}
		if err := repo.Reset(ctx, rest[0], mode); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "stash":
		args := rest
		includeUntracked := false
		if len(args) > 0 && (args[0] == "--untracked" || args[0] == "-u") {
			includeUntracked = true
			args = args[1:]
		}
		if err := repo.StashPush(ctx, strings.Join(args, " "), includeUntracked); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "stash-list":
		if len(rest) != 0 {
			return fail(out, "usage", "stash-list não aceita argumentos")
		}
		stashes, err := repo.StashList(ctx)
		if err != nil {
			return failGit(out, err)
		}
		return ok(out, stashes)

	case "stash-pop", "stash-apply", "stash-drop":
		index, err := parseStashIndex(rest)
		if err != nil {
			return fail(out, "usage", "uso: gitcontrol --json "+cmd+" [index]")
		}
		switch cmd {
		case "stash-pop":
			err = repo.StashPop(ctx, index)
		case "stash-apply":
			err = repo.StashApply(ctx, index)
		default:
			err = repo.StashDrop(ctx, index)
		}
		if err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "tags":
		if len(rest) != 0 {
			return fail(out, "usage", "tags não aceita argumentos")
		}
		tags, err := repo.Tags(ctx)
		if err != nil {
			return failGit(out, err)
		}
		return ok(out, tags)

	case "tag-create":
		if len(rest) == 0 {
			return fail(out, "usage", "uso: gitcontrol --json tag-create <name> [message...]")
		}
		if err := repo.CreateTag(ctx, rest[0], strings.Join(rest[1:], " ")); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	case "tag-delete":
		if len(rest) != 1 {
			return fail(out, "usage", "uso: gitcontrol --json tag-delete <name>")
		}
		if err := repo.DeleteTag(ctx, rest[0]); err != nil {
			return failGit(out, err)
		}
		return ok(out, nil)

	default:
		return fail(out, "usage", fmt.Sprintf("comando desconhecido: %s", cmd))
	}
}

// parseStashIndex lê um índice opcional (default 0).
func parseStashIndex(args []string) (int, error) {
	if len(args) == 0 {
		return 0, nil
	}
	if len(args) != 1 {
		return 0, fmt.Errorf("índice de stash inválido")
	}
	index, err := strconv.Atoi(args[0])
	if err != nil || index < 0 {
		return 0, fmt.Errorf("índice de stash inválido: %q", args[0])
	}
	return index, nil
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
