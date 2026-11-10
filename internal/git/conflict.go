package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Operation identifica uma operação multi-etapas em andamento.
type Operation string

const (
	OperationNone       Operation = ""
	OperationMerge      Operation = "merge"
	OperationCherryPick Operation = "cherry-pick"
	OperationRebase     Operation = "rebase"
)

// CurrentOperation descobre se existe merge/cherry-pick em andamento.
func (r *Repo) CurrentOperation(ctx context.Context) (Operation, error) {
	dir, err := r.GitDir(ctx)
	if err != nil {
		return OperationNone, err
	}
	return detectOperation(dir), nil
}

func detectOperation(gitDir string) Operation {
	for _, dir := range []string{"rebase-merge", "rebase-apply"} {
		if info, err := os.Stat(filepath.Join(gitDir, dir)); err == nil && info.IsDir() {
			return OperationRebase
		}
	}

	for _, candidate := range []struct {
		file string
		op   Operation
	}{
		{"MERGE_HEAD", OperationMerge},
		{"CHERRY_PICK_HEAD", OperationCherryPick},
	} {
		if _, err := os.Stat(filepath.Join(gitDir, candidate.file)); err == nil {
			return candidate.op
		}
	}
	return OperationNone
}

// Resolve marca arquivos conflitados como resolvidos (git add).
func (r *Repo) Resolve(ctx context.Context, paths []string) error {
	return r.Stage(ctx, paths)
}

// Abort desfaz a operação em andamento.
func (r *Repo) Abort(ctx context.Context) error {
	op, err := r.CurrentOperation(ctx)
	if err != nil {
		return err
	}

	switch op {
	case OperationMerge:
		return r.runAction(ctx, "merge", "--abort")
	case OperationCherryPick:
		return r.runAction(ctx, "cherry-pick", "--abort")
	case OperationRebase:
		return r.runAction(ctx, "rebase", "--abort")
	default:
		return fmt.Errorf("nenhuma operação para abortar")
	}
}

// Continue conclui a operação em andamento depois da resolução.
func (r *Repo) Continue(ctx context.Context) error {
	op, err := r.CurrentOperation(ctx)
	if err != nil {
		return err
	}

	switch op {
	case OperationMerge:
		return r.runAction(ctx, "commit", "--no-edit")
	case OperationCherryPick:
		return r.runAction(ctx, "cherry-pick", "--continue")
	case OperationRebase:
		return r.runAction(ctx, "rebase", "--continue")
	default:
		return fmt.Errorf("nenhuma operação para continuar")
	}
}

func (r *Repo) runAction(ctx context.Context, args ...string) error {
	stdout, stderr, err := r.run(ctx, args...)
	if err != nil {
		return gitError("git "+args[0], stdout, stderr, err)
	}
	return nil
}
