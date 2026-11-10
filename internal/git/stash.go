package git

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Stash representa uma entrada de `git stash list`.
type Stash struct {
	Index   int    `json:"index"`
	Ref     string `json:"ref"`
	Message string `json:"message"`
}

// StashPush guarda as mudanças locais. message vazio usa a mensagem do git.
func (r *Repo) StashPush(ctx context.Context, message string, includeUntracked bool) error {
	args := []string{"stash", "push"}
	if includeUntracked {
		args = append(args, "--include-untracked")
	}
	if message = strings.TrimSpace(message); message != "" {
		args = append(args, "-m", message)
	}

	stdout, stderr, err := r.run(ctx, args...)
	if err != nil {
		return gitError("git stash push", stdout, stderr, err)
	}
	return nil
}

// StashList lista os stashes do mais recente para o mais antigo.
func (r *Repo) StashList(ctx context.Context) ([]Stash, error) {
	stdout, stderr, err := r.run(ctx, "stash", "list", "--pretty=format:%gd%x1f%gs")
	if err != nil {
		return nil, fmt.Errorf("git stash list: %w (stderr: %s)", err, stderr)
	}

	if len(stdout) == 0 {
		return []Stash{}, nil
	}

	stashes := []Stash{}
	for _, line := range strings.Split(strings.TrimRight(string(stdout), "\n"), "\n") {
		parts := strings.SplitN(line, logSeparator, 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("linha de stash inválida: %q", line)
		}
		index, err := stashIndex(parts[0])
		if err != nil {
			return nil, err
		}
		stashes = append(stashes, Stash{
			Index:   index,
			Ref:     parts[0],
			Message: parts[1],
		})
	}
	return stashes, nil
}

func stashIndex(ref string) (int, error) {
	open := strings.IndexByte(ref, '{')
	close := strings.IndexByte(ref, '}')
	if open < 0 || close <= open {
		return 0, fmt.Errorf("ref de stash inválida: %q", ref)
	}
	index, err := strconv.Atoi(ref[open+1 : close])
	if err != nil {
		return 0, fmt.Errorf("índice de stash inválido %q: %w", ref, err)
	}
	return index, nil
}

// StashPop aplica e remove o stash.
func (r *Repo) StashPop(ctx context.Context, index int) error {
	return r.stashAction(ctx, "pop", index)
}

// StashApply aplica o stash mantendo-o na lista.
func (r *Repo) StashApply(ctx context.Context, index int) error {
	return r.stashAction(ctx, "apply", index)
}

// StashDrop remove o stash sem aplicar.
func (r *Repo) StashDrop(ctx context.Context, index int) error {
	return r.stashAction(ctx, "drop", index)
}

func (r *Repo) stashAction(ctx context.Context, verb string, index int) error {
	if index < 0 {
		return fmt.Errorf("git stash %s: índice inválido %d", verb, index)
	}
	ref := fmt.Sprintf("stash@{%d}", index)
	stdout, stderr, err := r.run(ctx, "stash", verb, ref)
	if err != nil {
		return gitError("git stash "+verb+" "+ref, stdout, stderr, err)
	}
	return nil
}
