package git

import (
	"context"
	"fmt"
)

// Modos aceitos por Reset.
const (
	ResetSoft  = "soft"
	ResetMixed = "mixed"
	ResetHard  = "hard"
)

// Reset move o HEAD para rev. mode vazio usa mixed.
func (r *Repo) Reset(ctx context.Context, rev, mode string) error {
	switch mode {
	case "", ResetMixed:
		mode = ResetMixed
	case ResetSoft, ResetHard:
	default:
		return fmt.Errorf("modo de reset inválido: %q", mode)
	}

	if rev == "" {
		rev = "HEAD"
	}

	stdout, stderr, err := r.run(ctx, "reset", "--"+mode, rev)
	if err != nil {
		return gitError("git reset --"+mode+" "+rev, stdout, stderr, err)
	}
	return nil
}

// Discard descarta mudanças dos paths: restaura tracked e remove untracked.
func (r *Repo) Discard(ctx context.Context, paths []string) error {
	if len(paths) == 0 {
		return fmt.Errorf("git restore: nenhum path informado")
	}

	for _, path := range paths {
		if path == "" {
			continue
		}
		if r.isTracked(ctx, path) {
			stdout, stderr, err := r.run(ctx, "restore", "--staged", "--worktree", "--", path)
			if err != nil {
				return gitError("git restore "+path, stdout, stderr, err)
			}
			continue
		}

		stdout, stderr, err := r.run(ctx, "clean", "-fd", "--", path)
		if err != nil {
			return gitError("git clean "+path, stdout, stderr, err)
		}
	}
	return nil
}
