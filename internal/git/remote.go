package git

import (
	"context"
	"fmt"
	"strings"
)

// Remote é um remoto configurado no repositório.
type Remote struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Remotes lista os remotos configurados (fetch), sem duplicar entradas.
func (r *Repo) Remotes(ctx context.Context) ([]Remote, error) {
	stdout, stderr, err := r.run(ctx, "remote", "-v")
	if err != nil {
		return nil, fmt.Errorf("git remote: %w (stderr: %s)", err, stderr)
	}

	seen := map[string]bool{}
	remotes := []Remote{}
	for _, line := range strings.Split(strings.TrimRight(string(stdout), "\n"), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || seen[fields[0]] {
			continue
		}
		seen[fields[0]] = true
		remotes = append(remotes, Remote{Name: fields[0], URL: fields[1]})
	}
	return remotes, nil
}

// Fetch atualiza as refs remotas. Sem remote, usa o padrão do branch atual.
func (r *Repo) Fetch(ctx context.Context, remote string) error {
	args := []string{"fetch", "--prune"}
	if remote = strings.TrimSpace(remote); remote != "" {
		args = append(args, remote)
	}

	stdout, stderr, err := r.run(ctx, args...)
	if err != nil {
		return gitError("git fetch", stdout, stderr, err)
	}
	return nil
}

// Pull incorpora o remoto com merge. Conflitos ficam no repositório para
// abort/continue, igual ao Merge.
func (r *Repo) Pull(ctx context.Context, remote, branch string) error {
	args := []string{"pull", "--no-edit"}
	if remote = strings.TrimSpace(remote); remote != "" {
		args = append(args, remote)
		if branch = strings.TrimSpace(branch); branch != "" {
			args = append(args, branch)
		}
	}

	stdout, stderr, err := r.run(ctx, args...)
	if err != nil {
		return gitError("git pull", stdout, stderr, err)
	}
	return nil
}

// Push envia o branch atual. Sem upstream configurado, cria o tracking com
// -u no remote informado (origin por padrão).
func (r *Repo) Push(ctx context.Context, remote, branch string) error {
	remote = strings.TrimSpace(remote)
	branch = strings.TrimSpace(branch)

	if branch == "" {
		var err error
		branch, err = r.currentBranch(ctx)
		if err != nil {
			return err
		}
	}
	if remote == "" {
		remote = "origin"
	}

	status, err := r.Status(ctx)
	if err != nil {
		return err
	}

	args := []string{"push"}
	if status.Upstream == "" {
		args = append(args, "-u", remote, branch)
	} else {
		args = append(args, remote, branch)
	}

	stdout, stderr, runErr := r.run(ctx, args...)
	if runErr != nil {
		return gitError("git push", stdout, stderr, runErr)
	}
	return nil
}

// currentBranch devolve o branch atual; falha em HEAD destacado.
func (r *Repo) currentBranch(ctx context.Context) (string, error) {
	stdout, stderr, err := r.run(ctx, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return "", fmt.Errorf("branch atual: %w (stderr: %s)", err, stderr)
	}
	branch := strings.TrimSpace(string(stdout))
	if branch == "" {
		return "", fmt.Errorf("branch atual vazia")
	}
	return branch, nil
}
