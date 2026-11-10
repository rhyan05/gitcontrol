package git

import (
	"context"
	"fmt"
	"strings"
)

// CreateBranch cria uma branch. startPoint vazio usa o HEAD atual.
func (r *Repo) CreateBranch(ctx context.Context, name, startPoint string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("git branch: nome vazio")
	}

	args := []string{"branch", name}
	if startPoint = strings.TrimSpace(startPoint); startPoint != "" {
		args = append(args, startPoint)
	}

	stdout, stderr, err := r.run(ctx, args...)
	if err != nil {
		return gitError("git branch "+name, stdout, stderr, err)
	}
	return nil
}

// DeleteBranch remove uma branch local. force usa -D em vez de -d.
func (r *Repo) DeleteBranch(ctx context.Context, name string, force bool) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("git branch -d: nome vazio")
	}

	flag := "-d"
	if force {
		flag = "-D"
	}

	stdout, stderr, err := r.run(ctx, "branch", flag, name)
	if err != nil {
		return gitError("git branch "+flag+" "+name, stdout, stderr, err)
	}
	return nil
}

// RenameBranch renomeia old para new. old vazio renomeia a branch atual.
func (r *Repo) RenameBranch(ctx context.Context, old, new string) error {
	new = strings.TrimSpace(new)
	if new == "" {
		return fmt.Errorf("git branch -m: novo nome vazio")
	}

	args := []string{"branch", "-m"}
	if old = strings.TrimSpace(old); old != "" {
		args = append(args, old)
	}
	args = append(args, new)

	stdout, stderr, err := r.run(ctx, args...)
	if err != nil {
		return gitError("git branch -m "+new, stdout, stderr, err)
	}
	return nil
}
